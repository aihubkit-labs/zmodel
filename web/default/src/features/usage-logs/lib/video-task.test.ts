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
  buildPublicVideoContentURL,
  buildPublicVideoDownloadURL,
} from './video-task'

describe('public video URLs', () => {
  test('uses the platform content endpoint for copy and playback', () => {
    assert.equal(
      buildPublicVideoContentURL(
        'task_5lwBWioFgvZHkMxv6AteO1stPpQPitVg',
        'https://api.example.com'
      ),
      'https://api.example.com/v1/videos/task_5lwBWioFgvZHkMxv6AteO1stPpQPitVg/content'
    )
  })

  test('uses the same content endpoint for downloads', () => {
    assert.equal(
      buildPublicVideoDownloadURL('task id/1'),
      '/v1/videos/task%20id%2F1/content?download_name=task%20id%2F1'
    )
  })
})
