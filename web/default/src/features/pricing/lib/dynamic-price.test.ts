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

import type { PricingModel } from '../types'
import { formatMediaConditionSummary, inferMediaUnit } from './billing-expr'
import {
  formatDynamicMediaPrice,
  getDynamicPricingSummary,
} from './dynamic-price'

describe('dynamic media pricing display', () => {
  test('keeps reference count tiers structured for marketplace rendering', () => {
    const model = {
      billing_mode: 'tiered_expr',
      billing_expr:
        'v2:reference_video_count >= 1 && reference_video_count <= 10 ? tier("one_to_ten", usd(0.15 * units)) : tier("over_ten", usd(0.2 * units))',
    } as PricingModel

    const summary = getDynamicPricingSummary(model, { tokenUnit: 'M' })

    assert.ok(summary)
    assert.equal(summary.tierCount, 2)
    assert.equal(inferMediaUnit(summary.tiers), 'video')
    assert.ok(summary.tier?.mediaCondition)
    assert.equal(
      formatMediaConditionSummary(summary.tier.mediaCondition, (key) => key),
      'Reference video count 1–10'
    )
    const price = formatDynamicMediaPrice(
      summary.tier,
      'video',
      { tokenUnit: 'M' },
      {
        perImage: 'image',
        perVideo: 'video',
        perOutput: 'output',
        second: 'second',
      }
    )
    assert.match(price, /0\.15/)
    assert.match(price, /\/ video$/)
  })
})
