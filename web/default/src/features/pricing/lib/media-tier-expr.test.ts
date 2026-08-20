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

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  MEDIA_BILLING_FIXED_PLUS_SECOND,
  MEDIA_BILLING_PER_SECOND,
  MEDIA_BILLING_PER_TOTAL_TOKEN,
  MEDIA_BILLING_PER_UNIT,
  generateMediaExpr,
  tryParseMediaConfig,
  type MediaVisualConfig,
} from './media-tier-expr'

describe('media tier expression editor', () => {
  test('round-trips image tiers and keeps the last tier as fallback', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: '1K',
          conditions: [
            { variable: 'image_size_tier', operator: 'eq', value: '1K' },
          ],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.05,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: '4K',
          conditions: [
            { variable: 'image_size_tier', operator: 'eq', value: '4K' },
          ],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.15,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: '2K',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.125,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
      ],
    }

    const expr = generateMediaExpr(config)

    assert.equal(
      expr,
      'v2:image_size_tier == "1K" ? tier("1K", usd(0.05 * units)) : image_size_tier == "4K" ? tier("4K", usd(0.15 * units)) : tier("2K", usd(0.125 * units))'
    )
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('round-trips per-second and fixed-plus-second video prices', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: '720p',
          conditions: [
            { variable: 'resolution_tier', operator: 'eq', value: '720p' },
          ],
          billingMethod: MEDIA_BILLING_PER_SECOND,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0.025,
        },
        {
          label: '4K',
          conditions: [],
          billingMethod: MEDIA_BILLING_FIXED_PLUS_SECOND,
          unitPrice: 0,
          fixedPrice: 0.05,
          perSecondPrice: 0.04,
        },
      ],
    }

    assert.deepEqual(tryParseMediaConfig(generateMediaExpr(config)), config)
  })

  test('round-trips total-token settlement with duration reserve', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: 'without_reference',
          conditions: [
            {
              variable: 'reference_video_count',
              operator: 'eq',
              value: '0',
            },
          ],
          billingMethod: MEDIA_BILLING_PER_TOTAL_TOKEN,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0,
          totalTokenPrice: 70,
          reservePerSecond: 1.5,
        },
        {
          label: 'with_reference',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_TOTAL_TOKEN,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0,
          totalTokenPrice: 42,
          reservePerSecond: 3,
        },
      ],
    }

    const expr = generateMediaExpr(config)
    assert.equal(
      expr,
      'v3:reference_video_count == 0 ? tier("without_reference", deferred(total * 70, usd(1.5 * seconds * units))) : tier("with_reference", deferred(total * 42, usd(3 * seconds * units)))'
    )
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('round-trips reference media count comparisons', () => {
    const variables = [
      'reference_image_count',
      'reference_video_count',
      'reference_audio_count',
    ] as const

    for (const conditionVariable of variables) {
      const config: MediaVisualConfig = {
        tiers: [
          {
            label: 'with_reference',
            conditions: [
              { variable: conditionVariable, operator: 'lte', value: '10' },
            ],
            billingMethod: MEDIA_BILLING_PER_UNIT,
            unitPrice: 0.15,
            fixedPrice: 0,
            perSecondPrice: 0,
          },
          {
            label: 'base',
            conditions: [],
            billingMethod: MEDIA_BILLING_PER_UNIT,
            unitPrice: 0.2,
            fixedPrice: 0,
            perSecondPrice: 0,
          },
        ],
      }

      assert.deepEqual(tryParseMediaConfig(generateMediaExpr(config)), config)
    }
  })

  test('generates inclusive reference count ranges', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: 'none',
          conditions: [
            {
              variable: 'reference_video_count',
              operator: 'eq',
              value: '0',
            },
          ],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.1,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: 'one_to_ten',
          conditions: [
            {
              variable: 'reference_video_count',
              operator: 'range',
              value: '1',
              rangeEnd: '10',
            },
          ],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.15,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: 'over_ten',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.2,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
      ],
    }

    const expr = generateMediaExpr(config)

    assert.equal(
      expr,
      'v2:reference_video_count == 0 ? tier("none", usd(0.1 * units)) : reference_video_count >= 1 && reference_video_count <= 10 ? tier("one_to_ten", usd(0.15 * units)) : tier("over_ten", usd(0.2 * units))'
    )
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('combines resolution and reference video count conditions', () => {
    const priceTier = (
      label: string,
      resolution: string,
      operator: 'eq' | 'gt',
      count: string,
      unitPrice: number
    ): MediaVisualConfig['tiers'][number] => ({
      label,
      conditions: [
        {
          variable: 'resolution_tier' as const,
          operator: 'eq' as const,
          value: resolution,
        },
        {
          variable: 'reference_video_count' as const,
          operator,
          value: count,
        },
      ],
      billingMethod: MEDIA_BILLING_PER_UNIT,
      unitPrice,
      fixedPrice: 0,
      perSecondPrice: 0,
    })
    const config: MediaVisualConfig = {
      tiers: [
        priceTier('720p_with_reference', '720p', 'gt', '0', 0.12),
        priceTier('720p_without_reference', '720p', 'eq', '0', 0.08),
        priceTier('1080p_with_reference', '1080p', 'gt', '0', 0.18),
        priceTier('1080p_without_reference', '1080p', 'eq', '0', 0.14),
        {
          label: 'fallback',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.2,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
      ],
    }

    const expr = generateMediaExpr(config)

    assert.equal(
      expr,
      'v2:resolution_tier == "720p" && reference_video_count > 0 ? tier("720p_with_reference", usd(0.12 * units)) : resolution_tier == "720p" && reference_video_count == 0 ? tier("720p_without_reference", usd(0.08 * units)) : resolution_tier == "1080p" && reference_video_count > 0 ? tier("1080p_with_reference", usd(0.18 * units)) : resolution_tier == "1080p" && reference_video_count == 0 ? tier("1080p_without_reference", usd(0.14 * units)) : tier("fallback", usd(0.2 * units))'
    )
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('round-trips multiple time ranges in one timezone', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: 'peak',
          conditions: [
            {
              variable: 'time_range',
              timezone: 'Asia/Shanghai',
              ranges: [
                { start: '09:00', end: '11:59' },
                { start: '14:00', end: '17:59' },
              ],
            },
          ],
          billingMethod: MEDIA_BILLING_PER_SECOND,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0.32,
        },
        {
          label: 'off_peak',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_SECOND,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0.16,
        },
      ],
    }

    const expr = generateMediaExpr(config)

    assert.equal(
      expr,
      'v2:((hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai") >= 540 && hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai") <= 719) || (hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai") >= 840 && hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai") <= 1079)) ? tier("peak", usd(0.32 * seconds * units)) : tier("off_peak", usd(0.16 * seconds * units))'
    )
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('round-trips a time range that crosses midnight', () => {
    const config: MediaVisualConfig = {
      tiers: [
        {
          label: 'off_peak',
          conditions: [
            {
              variable: 'time_range',
              timezone: 'UTC',
              ranges: [{ start: '22:00', end: '06:00' }],
            },
          ],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.1,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: 'peak',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.2,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
      ],
    }

    const expr = generateMediaExpr(config)

    assert.match(expr, />= 1320 \|\| .* <= 360/)
    assert.deepEqual(tryParseMediaConfig(expr), config)
  })

  test('does not emit incomplete conditional tiers before the fallback', () => {
    const expr = generateMediaExpr({
      tiers: [
        {
          label: 'incomplete',
          conditions: [{ variable: 'quality', operator: 'eq', value: '   ' }],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.01,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: 'base',
          conditions: [],
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.08,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
      ],
    })

    assert.equal(expr, 'v2:tier("base", usd(0.08 * units))')
  })

  test('rejects advanced media expressions the visual editor cannot preserve', () => {
    assert.equal(
      tryParseMediaConfig('v2:tier("minimum", usd(0.02 * max(seconds, 5)))'),
      null
    )
  })
})
