/*
Copyright (C) 2025 QuantumNous

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

import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { mergeTaskLogProgress } from './taskLogProgress';

describe('task log polling', () => {
  test('merges only progress and status into existing rows', () => {
    const current = [
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
      },
    ];
    const refreshed = [
      {
        id: 1,
        task_id: 'task-1',
        progress: '80%',
        status: 'SUCCESS',
        fail_reason: 'must not replace current data',
        result_url: 'refreshed-result',
      },
    ];

    const merged = mergeTaskLogProgress(current, refreshed);

    assert.deepEqual(merged, [
      {
        ...current[0],
        progress: '80%',
        status: 'SUCCESS',
      },
      current[1],
    ]);
    assert.equal(merged[1], current[1]);
  });
});
