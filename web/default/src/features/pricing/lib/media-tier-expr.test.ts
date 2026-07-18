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
  MEDIA_BILLING_PER_UNIT,
  MEDIA_CONDITION_NONE,
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
          conditionVariable: 'image_size_tier',
          conditionValue: '1K',
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.05,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: '4K',
          conditionVariable: 'image_size_tier',
          conditionValue: '4K',
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.15,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: '2K',
          conditionVariable: MEDIA_CONDITION_NONE,
          conditionValue: '',
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
          conditionVariable: 'resolution_tier',
          conditionValue: '720p',
          billingMethod: MEDIA_BILLING_PER_SECOND,
          unitPrice: 0,
          fixedPrice: 0,
          perSecondPrice: 0.025,
        },
        {
          label: '4K',
          conditionVariable: MEDIA_CONDITION_NONE,
          conditionValue: '',
          billingMethod: MEDIA_BILLING_FIXED_PLUS_SECOND,
          unitPrice: 0,
          fixedPrice: 0.05,
          perSecondPrice: 0.04,
        },
      ],
    }

    assert.deepEqual(tryParseMediaConfig(generateMediaExpr(config)), config)
  })

  test('does not emit incomplete conditional tiers before the fallback', () => {
    const expr = generateMediaExpr({
      tiers: [
        {
          label: 'incomplete',
          conditionVariable: 'quality',
          conditionValue: '   ',
          billingMethod: MEDIA_BILLING_PER_UNIT,
          unitPrice: 0.01,
          fixedPrice: 0,
          perSecondPrice: 0,
        },
        {
          label: 'base',
          conditionVariable: MEDIA_CONDITION_NONE,
          conditionValue: '',
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
