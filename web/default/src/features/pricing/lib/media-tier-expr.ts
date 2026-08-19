/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import {
  parseTiersFromExpr,
  type MediaCondition as ParsedMediaCondition,
  type MediaConditionOperator as BillingMediaConditionOperator,
  type MediaConditionVariable as ParsedMediaConditionVariable,
} from './billing-expr'

export type MediaConditionOperator = BillingMediaConditionOperator

export const MEDIA_BILLING_PER_UNIT = 'per_unit'
export const MEDIA_BILLING_PER_SECOND = 'per_second'
export const MEDIA_BILLING_FIXED_PLUS_SECOND = 'fixed_plus_second'
export const MEDIA_BILLING_PER_TOTAL_TOKEN = 'per_total_token'

export const MEDIA_CONDITION_EQ = 'eq'
export const MEDIA_CONDITION_LT = 'lt'
export const MEDIA_CONDITION_LTE = 'lte'
export const MEDIA_CONDITION_GT = 'gt'
export const MEDIA_CONDITION_GTE = 'gte'
export const MEDIA_CONDITION_RANGE = 'range'

export type MediaBillingMethod =
  | typeof MEDIA_BILLING_PER_UNIT
  | typeof MEDIA_BILLING_PER_SECOND
  | typeof MEDIA_BILLING_FIXED_PLUS_SECOND
  | typeof MEDIA_BILLING_PER_TOTAL_TOKEN

export type MediaConditionVariable = ParsedMediaConditionVariable

export type MediaTierCondition = ParsedMediaCondition

export type MediaTier = {
  label: string
  conditions: MediaTierCondition[]
  billingMethod: MediaBillingMethod
  unitPrice: number
  fixedPrice: number
  perSecondPrice: number
  totalTokenPrice?: number
  reservePerSecond?: number
}

export function isReferenceCountCondition(
  variable: MediaConditionVariable
): variable is
  | 'reference_image_count'
  | 'reference_video_count'
  | 'reference_audio_count' {
  return variable.startsWith('reference_') && variable.endsWith('_count')
}

export type MediaVisualConfig = {
  tiers: MediaTier[]
}

export function createDefaultMediaTier(index = 0): MediaTier {
  return {
    label: index === 0 ? 'base' : `tier_${index + 1}`,
    conditions: [],
    billingMethod: MEDIA_BILLING_PER_UNIT,
    unitPrice: 0,
    fixedPrice: 0,
    perSecondPrice: 0,
    totalTokenPrice: 0,
    reservePerSecond: 0,
  }
}

export function createDefaultMediaConfig(): MediaVisualConfig {
  return { tiers: [createDefaultMediaTier()] }
}

function finitePrice(value: unknown): number {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? number : 0
}

function escapeExprString(value: string): string {
  return value.replaceAll('\\', '\\\\').replaceAll('"', '\\"')
}

function parseReferenceCount(value: string): number | null {
  const normalized = value.trim()
  if (!/^\d+$/.test(normalized)) return null
  const count = Number(normalized)
  return Number.isSafeInteger(count) ? count : null
}

function buildMediaCondition(condition: MediaTierCondition): string | null {
  if (!isReferenceCountCondition(condition.variable)) {
    const value = condition.value.trim()
    if (!value) return null
    return `${condition.variable} == "${escapeExprString(value)}"`
  }

  const value = parseReferenceCount(condition.value)
  if (value == null) return null
  const operator = condition.operator || MEDIA_CONDITION_EQ
  if (operator === MEDIA_CONDITION_RANGE) {
    const rangeEnd = parseReferenceCount(condition.rangeEnd || '')
    if (rangeEnd == null || value > rangeEnd) return null
    return `${condition.variable} >= ${value} && ${condition.variable} <= ${rangeEnd}`
  }
  const symbols: Record<Exclude<MediaConditionOperator, 'range'>, string> = {
    eq: '==',
    lt: '<',
    lte: '<=',
    gt: '>',
    gte: '>=',
  }
  return `${condition.variable} ${symbols[operator]} ${value}`
}

function buildMediaTierCondition(tier: MediaTier): string | null {
  if (tier.conditions.length === 0) return null
  const conditions = tier.conditions.map(buildMediaCondition)
  if (conditions.some((condition) => condition == null)) return null
  return conditions.join(' && ')
}

function buildMediaCost(tier: MediaTier): string {
  if (tier.billingMethod === MEDIA_BILLING_PER_TOTAL_TOKEN) {
    return `deferred(total * ${finitePrice(tier.totalTokenPrice)}, usd(${finitePrice(tier.reservePerSecond)} * seconds * units))`
  }
  if (tier.billingMethod === MEDIA_BILLING_PER_SECOND) {
    return `usd(${finitePrice(tier.perSecondPrice)} * seconds * units)`
  }
  if (tier.billingMethod === MEDIA_BILLING_FIXED_PLUS_SECOND) {
    return `usd((${finitePrice(tier.fixedPrice)} + ${finitePrice(tier.perSecondPrice)} * seconds) * units)`
  }
  return `usd(${finitePrice(tier.unitPrice)} * units)`
}

export function generateMediaExpr(config: MediaVisualConfig): string {
  const sourceTiers =
    config.tiers.length > 0 ? config.tiers : [createDefaultMediaTier()]
  const fallback = sourceTiers.at(-1) || createDefaultMediaTier()
  const conditionalTiers = sourceTiers.slice(0, -1).flatMap((tier) => {
    const condition = buildMediaTierCondition(tier)
    return condition ? [{ tier, condition }] : []
  })
  const tiers = [...conditionalTiers.map(({ tier }) => tier), fallback]
  const parts = tiers.map((tier, index) => {
    const label = escapeExprString(tier.label || `tier_${index + 1}`)
    const body = `tier("${label}", ${buildMediaCost(tier)})`
    if (index < tiers.length - 1) {
      return `${conditionalTiers[index].condition} ? ${body}`
    }
    return body
  })
  const version = tiers.some(
    (tier) => tier.billingMethod === MEDIA_BILLING_PER_TOTAL_TOKEN
  )
    ? 'v3'
    : 'v2'
  return `${version}:${parts.join(' : ')}`
}

export function isMediaBillingExpr(expr: string | null | undefined): boolean {
  return Boolean(expr && /^v[23]:/.test(expr) && /\busd\s*\(/.test(expr))
}

export function tryParseMediaConfig(
  expr: string | null | undefined
): MediaVisualConfig | null {
  if (!isMediaBillingExpr(expr)) return null
  const parsed = parseTiersFromExpr(expr || '')
  if (parsed.length === 0 || parsed.some((tier) => !tier.mediaPricing)) {
    return null
  }
  if (
    parsed
      .slice(0, -1)
      .some(
        (tier) =>
          tier.mediaConditions.length === 0 || tier.conditions.length > 0
      )
  ) {
    return null
  }
  return {
    tiers: parsed.map((tier) => {
      const pricing = tier.mediaPricing
      return {
        label: tier.label,
        conditions: tier.mediaConditions,
        billingMethod: pricing?.method || MEDIA_BILLING_PER_UNIT,
        unitPrice: finitePrice(pricing?.unitPrice),
        fixedPrice: finitePrice(pricing?.fixedPrice),
        perSecondPrice: finitePrice(pricing?.perSecondPrice),
        ...(pricing?.method === MEDIA_BILLING_PER_TOTAL_TOKEN
          ? {
              totalTokenPrice: finitePrice(pricing.totalTokenPrice),
              reservePerSecond: finitePrice(pricing.reservePerSecond),
            }
          : {}),
      }
    }),
  }
}
