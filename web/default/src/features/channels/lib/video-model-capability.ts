import type { VideoModelCapabilityTemplate } from '../types'
/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { VideoModelCapabilityFormValue } from './channel-form'

export function defaultVideoModelCapability(
  model = ''
): VideoModelCapabilityFormValue {
  return {
    model,
    resolutions: [],
    ratios: [],
    resolution_mappings: {},
    size_mappings: {},
    ratio_required: false,
    min_reference_images: 0,
    max_reference_images: 0,
    min_reference_videos: 0,
    max_reference_videos: 0,
    min_reference_audios: 0,
    max_reference_audios: 0,
    max_reference_media_count: undefined,
    supports_duration: true,
    duration_required: true,
    allowed_duration_seconds: [],
    supports_generate_audio: false,
    generate_audio_required: false,
    supports_first_frame: false,
    first_frame_required: false,
    supports_last_frame: false,
    last_frame_required: false,
    last_frame_requires_first_frame: false,
    reference_images_incompatible_with_frames: false,
    audio_reference_requires_visual_reference: false,
    reference_media_requires_visual_reference: false,
    reference_media_incompatible_with_frames: false,
    supports_seed: false,
    supports_watermark: false,
    auto_reference_mode: false,
    frames_as_reference_images: false,
    omit_parameters: [],
    fixed_parameters: {},
  }
}

export function videoModelCapabilityFromTemplate(
  template: VideoModelCapabilityTemplate,
  model: string
): VideoModelCapabilityFormValue {
  return {
    ...defaultVideoModelCapability(model),
    ...template.capability,
    model,
    resolutions: [...(template.capability.resolutions || [])],
    ratios: [...(template.capability.ratios || [])],
    resolution_mappings: {
      ...template.capability.resolution_mappings,
    },
    size_mappings: { ...template.capability.size_mappings },
    allowed_duration_seconds: [
      ...(template.capability.allowed_duration_seconds || []),
    ],
    omit_parameters: [...(template.capability.omit_parameters || [])],
    fixed_parameters: { ...template.capability.fixed_parameters },
  }
}
