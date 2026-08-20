/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  CACHE_MODE_GENERIC,
  evalExprLocally,
  generateExprFromVisualConfig,
  tryParseVisualConfig,
  type ExtraTokenValues,
  type VisualConfig,
} from './tier-expr'

const noExtraTokens: ExtraTokenValues = {
  cacheReadTokens: 0,
  cacheCreateTokens: 0,
  cacheCreate1hTokens: 0,
  imageTokens: 0,
  imageOutputTokens: 0,
  audioInputTokens: 0,
  audioOutputTokens: 0,
}

describe('visual tier time-range conditions', () => {
  test('round-trips an independent time tier with token conditions', () => {
    const config: VisualConfig = {
      tiers: [
        {
          label: 'peak_short',
          conditions: [
            {
              var: 'time_range',
              timezone: 'Asia/Shanghai',
              ranges: [
                { start: '09:00', end: '11:59' },
                { start: '14:00', end: '17:59' },
              ],
            },
            { var: 'len', op: '<=', value: 200000 },
          ],
          input_unit_cost: 3,
          output_unit_cost: 15,
          cache_mode: CACHE_MODE_GENERIC,
        },
        {
          label: 'off_peak',
          conditions: [],
          input_unit_cost: 2,
          output_unit_cost: 10,
          cache_mode: CACHE_MODE_GENERIC,
        },
      ],
    }

    const expr = generateExprFromVisualConfig(config)
    assert.match(expr, /hour\("Asia\/Shanghai"\)/)
    assert.match(expr, /&& len <= 200000 \? tier\("peak_short"/)

    const parsed = tryParseVisualConfig(expr)
    assert.ok(parsed)
    assert.deepEqual(parsed.tiers[0].conditions, config.tiers[0].conditions)
    assert.equal(parsed.tiers[0].input_unit_cost, 3)
    assert.equal(parsed.tiers[1].conditions.length, 0)
  })

  test('supports a cross-midnight independent price tier', () => {
    const config: VisualConfig = {
      tiers: [
        {
          label: 'night',
          conditions: [
            {
              var: 'time_range',
              timezone: 'UTC',
              ranges: [{ start: '22:00', end: '06:00' }],
            },
          ],
          input_unit_cost: 1,
          output_unit_cost: 2,
          cache_mode: CACHE_MODE_GENERIC,
        },
        {
          label: 'day',
          conditions: [],
          input_unit_cost: 3,
          output_unit_cost: 4,
          cache_mode: CACHE_MODE_GENERIC,
        },
      ],
    }

    const expr = generateExprFromVisualConfig(config)
    assert.match(expr, />= 1320 \|\| .* <= 360/)
    assert.deepEqual(
      tryParseVisualConfig(expr)?.tiers[0].conditions,
      config.tiers[0].conditions
    )
  })

  test('uses the fixed preview time to select the independent tier', () => {
    const expr = generateExprFromVisualConfig({
      tiers: [
        {
          label: 'peak',
          conditions: [
            {
              var: 'time_range',
              timezone: 'Asia/Shanghai',
              ranges: [{ start: '09:00', end: '11:59' }],
            },
          ],
          input_unit_cost: 3,
          output_unit_cost: 15,
          cache_mode: CACHE_MODE_GENERIC,
        },
        {
          label: 'off_peak',
          conditions: [],
          input_unit_cost: 2,
          output_unit_cost: 10,
          cache_mode: CACHE_MODE_GENERIC,
        },
      ],
    })

    const peak = evalExprLocally(
      expr,
      100,
      10,
      noExtraTokens,
      new Date('2026-08-20T02:00:00Z')
    )
    const offPeak = evalExprLocally(
      expr,
      100,
      10,
      noExtraTokens,
      new Date('2026-08-20T12:00:00Z')
    )

    assert.equal(peak.matchedTier, 'peak')
    assert.equal(peak.cost, 450)
    assert.equal(offPeak.matchedTier, 'off_peak')
    assert.equal(offPeak.cost, 300)
  })
})
