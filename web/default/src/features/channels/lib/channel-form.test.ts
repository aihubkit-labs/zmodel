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
  channelFormSchema,
  transformFormDataToCreatePayload,
} from './channel-form'
import {
  defaultVideoModelCapability,
  videoModelCapabilityFromTemplate,
} from './video-model-capability'
import type { VideoModelCapabilityTemplate } from '../types'

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

describe('Video capability templates', () => {
  test('applies templates that omit empty resolutions', () => {
    const template = {
      model_id: 'gemini_omni_flash',
      capability: {
        ratios: ['16:9', '9:16'],
        allowed_duration_seconds: [10],
        default_duration_seconds: 10,
      },
    } as VideoModelCapabilityTemplate

    const capability = videoModelCapabilityFromTemplate(
      template,
      template.model_id
    )

    assert.deepEqual(capability.resolutions, [])
    assert.deepEqual(capability.ratios, ['16:9', '9:16'])
  })

  test('accepts Lingganya models with a duration range and no discrete durations', () => {
    const capability = {
      ...defaultVideoModelCapability('sd-2.0-vip'),
      resolutions: ['720p'],
      ratios: ['16:9', '9:16'],
      min_duration_seconds: 4,
      max_duration_seconds: 15,
      default_duration_seconds: 6,
      max_reference_images: 9,
      max_reference_videos: 3,
      max_reference_audios: 3,
      allowed_duration_seconds: [],
    }
    const result = channelFormSchema.safeParse({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'lingganya',
      models: 'sd-2.0-vip',
      video_protocol: 'lingganya_video',
      video_model_capabilities: [capability],
    })

    assert.equal(result.success, true)
  })
})
