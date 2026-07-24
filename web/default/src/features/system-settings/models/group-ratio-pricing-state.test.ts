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
  buildGroupPricingDrafts,
  reconcileGroupPricingDrafts,
  serializeGroupPricingDrafts,
} from './group-ratio-pricing-state'

describe('group ratio pricing drafts', () => {
  test('uses the selectable-group description for a selected group', () => {
    const rows = buildGroupPricingDrafts(
      {
        groupRatio: { default: 1 },
        userUsableGroups: { default: 'Updated selectable note' },
        groupDescriptions: { default: 'Older saved note' },
        topupGroupRatio: {},
      },
      () => 'group-1'
    )

    assert.equal(rows[0]?.description, 'Updated selectable note')
  })

  test('persists a description after an unselected group is saved and rebuilt', () => {
    const initialRows = buildGroupPricingDrafts(
      {
        groupRatio: { default: 1 },
        userUsableGroups: { default: 'Default group note' },
        groupDescriptions: {},
        topupGroupRatio: {},
      },
      () => 'group-1'
    )
    const unselectedRows = initialRows.map((row) => ({
      ...row,
      selectable: false,
    }))

    const savedMaps = serializeGroupPricingDrafts(unselectedRows)
    const rebuiltRows = buildGroupPricingDrafts(savedMaps, () => 'group-2')

    assert.deepEqual(savedMaps.userUsableGroups, {})
    assert.deepEqual(savedMaps.groupDescriptions, {
      default: 'Default group note',
    })
    assert.equal(rebuiltRows[0]?.selectable, false)
    assert.equal(rebuiltRows[0]?.description, 'Default group note')
  })

  test('accepts an updated description for an unselected group', () => {
    const reconciledRows = reconcileGroupPricingDrafts(
      [
        {
          _id: 'group-2',
          name: 'default',
          ratio: '1',
          topupRatio: '',
          selectable: false,
          description: 'Updated note',
        },
      ],
      [
        {
          _id: 'group-1',
          name: 'default',
          ratio: '1',
          topupRatio: '',
          selectable: false,
          description: 'Old note',
        },
      ]
    )

    assert.equal(reconciledRows[0]?._id, 'group-1')
    assert.equal(reconciledRows[0]?.description, 'Updated note')
  })
})
