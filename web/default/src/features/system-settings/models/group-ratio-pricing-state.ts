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

export type GroupPricingDraft = {
  _id: string
  name: string
  ratio: string
  topupRatio: string
  selectable: boolean
  description: string
}

export type GroupPricingMaps = {
  groupRatio: Record<string, number>
  userUsableGroups: Record<string, string>
  groupDescriptions: Record<string, string>
  topupGroupRatio: Record<string, number>
}

function normalizeRatio(value: unknown): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 1
}

export function buildGroupPricingDrafts(
  maps: GroupPricingMaps,
  createId: () => string
): GroupPricingDraft[] {
  const names = new Set([
    ...Object.keys(maps.groupRatio),
    ...Object.keys(maps.userUsableGroups),
    ...Object.keys(maps.groupDescriptions),
    ...Object.keys(maps.topupGroupRatio),
  ])

  return [...names].map((name) => {
    const selectable = Object.hasOwn(maps.userUsableGroups, name)
    return {
      _id: createId(),
      name,
      ratio: String(normalizeRatio(maps.groupRatio[name])),
      topupRatio: Object.hasOwn(maps.topupGroupRatio, name)
        ? String(maps.topupGroupRatio[name])
        : '',
      selectable,
      description: String(
        selectable
          ? maps.userUsableGroups[name]
          : (maps.groupDescriptions[name] ?? '')
      ),
    }
  })
}

export function serializeGroupPricingDrafts(
  rows: GroupPricingDraft[]
): GroupPricingMaps {
  const maps: GroupPricingMaps = {
    groupRatio: {},
    userUsableGroups: {},
    groupDescriptions: {},
    topupGroupRatio: {},
  }

  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue

    maps.groupRatio[name] = normalizeRatio(row.ratio)
    maps.groupDescriptions[name] = row.description
    if (row.selectable) {
      maps.userUsableGroups[name] = row.description
    }
    const topup = row.topupRatio.trim()
    if (topup !== '' && Number.isFinite(Number(topup))) {
      maps.topupGroupRatio[name] = Number(topup)
    }
  }

  return maps
}

export function reconcileGroupPricingDrafts(
  incomingRows: GroupPricingDraft[],
  currentRows: GroupPricingDraft[]
): GroupPricingDraft[] {
  const currentRowsByName = new Map(
    currentRows.map((row) => [row.name, row] as const)
  )

  return incomingRows.map((incomingRow) => {
    const currentRow = currentRowsByName.get(incomingRow.name)
    if (!currentRow) return incomingRow

    return {
      ...incomingRow,
      _id: currentRow._id,
    }
  })
}
