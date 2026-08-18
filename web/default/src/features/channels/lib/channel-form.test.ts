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
  CHANNEL_FORM_DEFAULT_VALUES,
  transformFormDataToCreatePayload,
} from './channel-form'
import { defaultVideoModelCapability } from './video-model-capability'

describe('GlobalAIOpc asset preparation setting', () => {
  test('serializes the asset preparation mode only when enabled for a model', () => {
    const capability = {
      ...defaultVideoModelCapability('sd_2.5_discount_v1'),
      resolutions: ['720p'],
      asset_preparation_mode: 'globalaiopc_seedance' as const,
    }
    const enabledPayload = transformFormDataToCreatePayload({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      video_protocol: 'globalaiopc',
      video_model_capabilities: [capability],
    })
    const enabledSettings = JSON.parse(String(enabledPayload.channel.setting))
    assert.equal(
      enabledSettings.video_model_capabilities['sd_2.5_discount_v1']
        .asset_preparation_mode,
      'globalaiopc_seedance'
    )

    const disabledPayload = transformFormDataToCreatePayload({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      video_protocol: 'globalaiopc',
      video_model_capabilities: [
        {
          ...capability,
          asset_preparation_mode: '',
        },
      ],
    })
    const disabledSettings = JSON.parse(String(disabledPayload.channel.setting))
    assert.equal(
      'asset_preparation_mode' in
        disabledSettings.video_model_capabilities['sd_2.5_discount_v1'],
      false
    )
  })
})
