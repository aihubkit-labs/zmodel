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
  type MediaConditionOperator as BillingMediaConditionOperator,
  type MediaConditionVariable as ParsedMediaConditionVariable,
} from './billing-expr'

export type MediaConditionOperator = BillingMediaConditionOperator

export const MEDIA_BILLING_PER_UNIT = 'per_unit'
export const MEDIA_BILLING_PER_SECOND = 'per_second'
export const MEDIA_BILLING_FIXED_PLUS_SECOND = 'fixed_plus_second'

export const MEDIA_CONDITION_NONE = 'none'
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

export type MediaConditionVariable =
  | typeof MEDIA_CONDITION_NONE
  | ParsedMediaConditionVariable

export type MediaTier = {
  label: string
  conditionVariable: MediaConditionVariable
  conditionValue: string
  conditionOperator?: MediaConditionOperator
  conditionRangeEnd?: string
  billingMethod: MediaBillingMethod
  unitPrice: number
  fixedPrice: number
  perSecondPrice: number
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
    conditionVariable: MEDIA_CONDITION_NONE,
    conditionValue: '',
    billingMethod: MEDIA_BILLING_PER_UNIT,
    unitPrice: 0,
    fixedPrice: 0,
    perSecondPrice: 0,
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

function buildMediaCondition(tier: MediaTier): string | null {
  if (tier.conditionVariable === MEDIA_CONDITION_NONE) return null
  if (!isReferenceCountCondition(tier.conditionVariable)) {
    const value = tier.conditionValue.trim()
    if (!value) return null
    return `${tier.conditionVariable} == "${escapeExprString(value)}"`
  }

  const value = parseReferenceCount(tier.conditionValue)
  if (value == null) return null
  const operator = tier.conditionOperator || MEDIA_CONDITION_EQ
  if (operator === MEDIA_CONDITION_RANGE) {
    const rangeEnd = parseReferenceCount(tier.conditionRangeEnd || '')
    if (rangeEnd == null || value > rangeEnd) return null
    return `${tier.conditionVariable} >= ${value} && ${tier.conditionVariable} <= ${rangeEnd}`
  }
  const symbols: Record<Exclude<MediaConditionOperator, 'range'>, string> = {
    eq: '==',
    lt: '<',
    lte: '<=',
    gt: '>',
    gte: '>=',
  }
  return `${tier.conditionVariable} ${symbols[operator]} ${value}`
}

function buildMediaCost(tier: MediaTier): string {
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
    const condition = buildMediaCondition(tier)
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
  return `v2:${parts.join(' : ')}`
}

export function isMediaBillingExpr(expr: string | null | undefined): boolean {
  return Boolean(expr && expr.startsWith('v2:') && /\busd\s*\(/.test(expr))
}

export function tryParseMediaConfig(
  expr: string | null | undefined
): MediaVisualConfig | null {
  if (!isMediaBillingExpr(expr)) return null
  const parsed = parseTiersFromExpr(expr || '')
  if (parsed.length === 0 || parsed.some((tier) => !tier.mediaPricing)) {
    return null
  }
  if (parsed.slice(0, -1).some((tier) => tier.mediaCondition == null)) {
    return null
  }
  return {
    tiers: parsed.map((tier) => ({
      label: tier.label,
      conditionVariable: tier.mediaCondition?.variable || MEDIA_CONDITION_NONE,
      conditionValue: tier.mediaCondition?.value || '',
      ...(tier.mediaCondition?.operator &&
      tier.mediaCondition.operator !== MEDIA_CONDITION_EQ
        ? { conditionOperator: tier.mediaCondition.operator }
        : {}),
      ...(tier.mediaCondition?.rangeEnd
        ? { conditionRangeEnd: tier.mediaCondition.rangeEnd }
        : {}),
      billingMethod: tier.mediaPricing?.method || MEDIA_BILLING_PER_UNIT,
      unitPrice: finitePrice(tier.mediaPricing?.unitPrice),
      fixedPrice: finitePrice(tier.mediaPricing?.fixedPrice),
      perSecondPrice: finitePrice(tier.mediaPricing?.perSecondPrice),
    })),
  }
}
