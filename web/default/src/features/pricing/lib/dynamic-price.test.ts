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
import { formatMediaConditionsSummary, inferMediaUnit } from './billing-expr'
import {
  formatDynamicMediaPrice,
  getDynamicPricingSummary,
} from './dynamic-price'

describe('dynamic media pricing display', () => {
  test('keeps reference count tiers structured for marketplace rendering', () => {
    const model = {
      billing_mode: 'tiered_expr',
      billing_expr:
        'v2:resolution_tier == "720p" && reference_video_count > 0 ? tier("720p_with_reference", usd(0.15 * units)) : tier("fallback", usd(0.2 * units))',
    } as PricingModel

    const summary = getDynamicPricingSummary(model, { tokenUnit: 'M' })

    assert.ok(summary)
    assert.equal(summary.tierCount, 2)
    assert.equal(inferMediaUnit(summary.tiers), 'video')
    assert.ok(summary.tier)
    assert.equal(
      formatMediaConditionsSummary(summary.tier.mediaConditions, (key) => key),
      'Video resolution tier = 720p · Reference video count > 0'
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

  test('renders total-token price and duration reserve', () => {
    const model = {
      billing_mode: 'tiered_expr',
      billing_expr:
        'v3:tier("base", deferred(total * 70, usd(1.5 * seconds * units)))',
    } as PricingModel
    const summary = getDynamicPricingSummary(model, { tokenUnit: 'M' })

    assert.ok(summary?.tier)
    const price = formatDynamicMediaPrice(
      summary.tier,
      'video',
      { tokenUnit: 'M' },
      {
        perImage: 'image',
        perVideo: 'video',
        perOutput: 'output',
        second: 'second',
        totalTokens: 'total tokens',
        reserve: 'Reserve',
      }
    )
    assert.match(price, /70/)
    assert.match(price, /1M total tokens/)
    assert.match(price, /Reserve/)
    assert.match(price, /1\.5/)
  })
})
