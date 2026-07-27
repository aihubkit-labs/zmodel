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

import { mergeTaskLogProgress } from './utils'

describe('task log polling', () => {
  test('merges only progress and status into the current page', () => {
    const current = {
      items: [
        {
          id: 1,
          task_id: 'task-1',
          progress: '20%',
          status: 'IN_PROGRESS',
          fail_reason: '',
          result_url: 'current-result',
        },
        {
          id: 2,
          task_id: 'task-2',
          progress: '50%',
          status: 'IN_PROGRESS',
          fail_reason: '',
          result_url: 'second-result',
        },
      ],
      total: 42,
      page: 3,
      page_size: 20,
    }
    const refreshed = {
      items: [
        {
          id: 1,
          task_id: 'task-1',
          progress: '80%',
          status: 'SUCCESS',
          fail_reason: 'must not replace current data',
          result_url: 'refreshed-result',
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
    }

    const merged = mergeTaskLogProgress(current, refreshed)

    assert.deepEqual(merged, {
      ...current,
      items: [
        {
          ...current.items[0],
          progress: '80%',
          status: 'SUCCESS',
        },
        current.items[1],
      ],
    })
    assert.equal(merged.items[1], current.items[1])
  })
})
