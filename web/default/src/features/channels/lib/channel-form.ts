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
import { z } from 'zod'

import {
  CHANNEL_STATUS,
  ERROR_MESSAGES,
  MODEL_FETCHABLE_TYPES,
} from '../constants'
import type { Channel } from '../types'
import {
  CHANNEL_TYPE_ADVANCED_CUSTOM,
  advancedCustomConfigUsesRelativeUpstreamPath,
  parseAdvancedCustomConfig,
  stringifyAdvancedCustomConfig,
  validateAdvancedCustomConfig,
} from './advanced-custom'

// ============================================================================
// Form Validation Schema
// ============================================================================

function parseOptionalJson(value: string | undefined): unknown {
  if (!value?.trim()) return undefined
  return JSON.parse(value)
}

function isJsonObjectValue(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isOptionalJsonObject(value: string | undefined): boolean {
  try {
    const parsed = parseOptionalJson(value)
    return parsed === undefined || isJsonObjectValue(parsed)
  } catch {
    return false
  }
}

function isOptionalModelMapping(value: string | undefined): boolean {
  try {
    const parsed = parseOptionalJson(value)
    if (parsed === undefined) return true
    if (!isJsonObjectValue(parsed)) return false
    return Object.values(parsed).every((item) => typeof item === 'string')
  } catch {
    return false
  }
}

function isOptionalStatusCodeMapping(value: string | undefined): boolean {
  try {
    const parsed = parseOptionalJson(value)
    if (parsed === undefined) return true
    if (!isJsonObjectValue(parsed)) return false
    return Object.entries(parsed).every(([from, to]) => {
      const fromCode = Number(from)
      const toCode = Number(to)
      return (
        Number.isInteger(fromCode) &&
        Number.isInteger(toCode) &&
        fromCode >= 100 &&
        fromCode <= 599 &&
        toCode >= 100 &&
        toCode <= 599
      )
    })
  } catch {
    return false
  }
}

function isCodexCredential(value: string | undefined): boolean {
  try {
    const parsed = parseOptionalJson(value)
    if (parsed === undefined) return true
    return (
      isJsonObjectValue(parsed) &&
      typeof parsed.access_token === 'string' &&
      parsed.access_token.trim().length > 0 &&
      typeof parsed.account_id === 'string' &&
      parsed.account_id.trim().length > 0
    )
  } catch {
    return false
  }
}

function isVertexJsonKey(value: string | undefined): boolean {
  try {
    const parsed = parseOptionalJson(value)
    if (parsed === undefined) return true
    if (Array.isArray(parsed)) {
      return parsed.every((item) => isJsonObjectValue(item))
    }
    return isJsonObjectValue(parsed)
  } catch {
    return false
  }
}

function addRequiredIssue(
  ctx: z.RefinementCtx,
  path: string,
  message: string
): void {
  ctx.addIssue({
    code: z.ZodIssueCode.custom,
    path: [path],
    message,
  })
}

export const MAX_VIDEO_REFERENCE_COUNT = 64
export const MAX_VIDEO_MODEL_RESOLUTIONS = 32
export const MAX_VIDEO_MODEL_RATIOS = 32
export const MAX_VIDEO_DURATION_SECONDS = 3600

const videoResolutionSchema = z.string().trim().min(1).max(64)
const videoRatioSchema = z.string().trim().min(1).max(64)
const videoDurationSchema = z
  .number()
  .int('Duration must be a whole number')
  .min(1, 'Duration must be between 1 and 3600 seconds')
  .max(
    MAX_VIDEO_DURATION_SECONDS,
    'Duration must be between 1 and 3600 seconds'
  )
const videoModelCapabilitySchema = z.object({
  model: z.string(),
  resolutions: z
    .array(videoResolutionSchema)
    .min(1, 'Configure at least one resolution')
    .max(MAX_VIDEO_MODEL_RESOLUTIONS),
  ratios: z.array(videoRatioSchema).max(MAX_VIDEO_MODEL_RATIOS).optional(),
  resolution_mappings: z.record(z.string(), z.string()).optional(),
  ratio_required: z.boolean().optional(),
  min_reference_images: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT).optional(),
  max_reference_images: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT),
  min_reference_videos: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT).optional(),
  max_reference_videos: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT),
  min_reference_audios: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT).optional(),
  max_reference_audios: z.number().int().min(0).max(MAX_VIDEO_REFERENCE_COUNT),
  supports_duration: z.boolean().optional(),
  duration_required: z.boolean().optional(),
  min_duration_seconds: videoDurationSchema.optional(),
  max_duration_seconds: videoDurationSchema.optional(),
  supports_generate_audio: z.boolean().optional(),
  generate_audio_required: z.boolean().optional(),
  supports_first_frame: z.boolean().optional(),
  first_frame_required: z.boolean().optional(),
  supports_last_frame: z.boolean().optional(),
  last_frame_required: z.boolean().optional(),
  last_frame_requires_first_frame: z.boolean().optional(),
  reference_images_incompatible_with_frames: z.boolean().optional(),
  audio_reference_requires_visual_reference: z.boolean().optional(),
  reference_media_incompatible_with_frames: z.boolean().optional(),
  supports_seed: z.boolean().optional(),
  min_seed: z.number().int().min(-1).max(2147483647).optional(),
  max_seed: z.number().int().min(-1).max(2147483647).optional(),
  supports_watermark: z.boolean().optional(),
  auto_reference_mode: z.boolean().optional(),
  reference_mode_for_references: z.string().optional(),
  reference_mode_for_frames: z.string().optional(),
  frames_as_reference_images: z.boolean().optional(),
  omit_parameters: z.array(z.string()).optional(),
  fixed_parameters: z.record(z.string(), z.unknown()).optional(),
})

export type VideoResolution = z.infer<typeof videoResolutionSchema>
export type VideoModelCapabilityFormValue = z.infer<
  typeof videoModelCapabilitySchema
>

function parseVideoCapabilityFlag(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function parseVideoModelCapabilities(
  value: unknown
): VideoModelCapabilityFormValue[] {
  if (!isJsonObjectValue(value)) return []

  const capabilities: VideoModelCapabilityFormValue[] = []
  for (const [model, rawCapability] of Object.entries(value)) {
    if (!isJsonObjectValue(rawCapability)) continue
    if (!Array.isArray(rawCapability.resolutions)) continue

    const normalizedResolutions = new Set<string>()
    const resolutions = rawCapability.resolutions
      .map((resolution) => String(resolution).trim().toLowerCase())
      .filter((resolution) => {
        if (!resolution || resolution.length > 64) return false
        if (normalizedResolutions.has(resolution)) return false
        normalizedResolutions.add(resolution)
        return true
      })
      .slice(0, MAX_VIDEO_MODEL_RESOLUTIONS)

    const normalizedRatios = new Set<string>()
    const ratios = Array.isArray(rawCapability.ratios)
      ? rawCapability.ratios
          .map((ratio) => String(ratio).trim().toLowerCase())
          .filter((ratio) => {
            if (!ratio || ratio.length > 64) return false
            if (normalizedRatios.has(ratio)) return false
            normalizedRatios.add(ratio)
            return true
          })
          .slice(0, MAX_VIDEO_MODEL_RATIOS)
      : undefined
    if (!model.trim() || resolutions.length === 0) continue

    capabilities.push({
      model: model.trim(),
      resolutions,
      ratios,
      resolution_mappings: isJsonObjectValue(rawCapability.resolution_mappings)
        ? Object.fromEntries(
            Object.entries(rawCapability.resolution_mappings)
              .filter(([, mapped]) => typeof mapped === 'string')
              .map(([resolution, mapped]) => [resolution, String(mapped)])
          )
        : undefined,
      ratio_required: parseVideoCapabilityFlag(rawCapability.ratio_required),
      min_reference_images:
        Number.isInteger(rawCapability.min_reference_images) &&
        Number(rawCapability.min_reference_images) >= 0 &&
        Number(rawCapability.min_reference_images) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.min_reference_images)
          : undefined,
      max_reference_images:
        Number.isInteger(rawCapability.max_reference_images) &&
        Number(rawCapability.max_reference_images) >= 0 &&
        Number(rawCapability.max_reference_images) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.max_reference_images)
          : 0,
      min_reference_videos:
        Number.isInteger(rawCapability.min_reference_videos) &&
        Number(rawCapability.min_reference_videos) >= 0 &&
        Number(rawCapability.min_reference_videos) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.min_reference_videos)
          : undefined,
      max_reference_videos:
        Number.isInteger(rawCapability.max_reference_videos) &&
        Number(rawCapability.max_reference_videos) >= 0 &&
        Number(rawCapability.max_reference_videos) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.max_reference_videos)
          : 0,
      min_reference_audios:
        Number.isInteger(rawCapability.min_reference_audios) &&
        Number(rawCapability.min_reference_audios) >= 0 &&
        Number(rawCapability.min_reference_audios) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.min_reference_audios)
          : undefined,
      max_reference_audios:
        Number.isInteger(rawCapability.max_reference_audios) &&
        Number(rawCapability.max_reference_audios) >= 0 &&
        Number(rawCapability.max_reference_audios) <= MAX_VIDEO_REFERENCE_COUNT
          ? Number(rawCapability.max_reference_audios)
          : 0,
      supports_duration: parseVideoCapabilityFlag(
        rawCapability.supports_duration
      ),
      duration_required: parseVideoCapabilityFlag(
        rawCapability.duration_required
      ),
      min_duration_seconds:
        Number.isInteger(rawCapability.min_duration_seconds) &&
        Number(rawCapability.min_duration_seconds) >= 1 &&
        Number(rawCapability.min_duration_seconds) <= MAX_VIDEO_DURATION_SECONDS
          ? Number(rawCapability.min_duration_seconds)
          : undefined,
      max_duration_seconds:
        Number.isInteger(rawCapability.max_duration_seconds) &&
        Number(rawCapability.max_duration_seconds) >= 1 &&
        Number(rawCapability.max_duration_seconds) <= MAX_VIDEO_DURATION_SECONDS
          ? Number(rawCapability.max_duration_seconds)
          : undefined,
      supports_generate_audio: parseVideoCapabilityFlag(
        rawCapability.supports_generate_audio
      ),
      generate_audio_required: parseVideoCapabilityFlag(
        rawCapability.generate_audio_required
      ),
      supports_first_frame: parseVideoCapabilityFlag(
        rawCapability.supports_first_frame
      ),
      first_frame_required: parseVideoCapabilityFlag(
        rawCapability.first_frame_required
      ),
      supports_last_frame: parseVideoCapabilityFlag(
        rawCapability.supports_last_frame
      ),
      last_frame_required: parseVideoCapabilityFlag(
        rawCapability.last_frame_required
      ),
      last_frame_requires_first_frame: parseVideoCapabilityFlag(
        rawCapability.last_frame_requires_first_frame
      ),
      reference_images_incompatible_with_frames: parseVideoCapabilityFlag(
        rawCapability.reference_images_incompatible_with_frames
      ),
      audio_reference_requires_visual_reference: parseVideoCapabilityFlag(
        rawCapability.audio_reference_requires_visual_reference
      ),
      reference_media_incompatible_with_frames: parseVideoCapabilityFlag(
        rawCapability.reference_media_incompatible_with_frames
      ),
      supports_seed: parseVideoCapabilityFlag(rawCapability.supports_seed),
      min_seed: Number.isInteger(rawCapability.min_seed)
        ? Number(rawCapability.min_seed)
        : undefined,
      max_seed: Number.isInteger(rawCapability.max_seed)
        ? Number(rawCapability.max_seed)
        : undefined,
      supports_watermark: parseVideoCapabilityFlag(
        rawCapability.supports_watermark
      ),
      auto_reference_mode: parseVideoCapabilityFlag(
        rawCapability.auto_reference_mode
      ),
      reference_mode_for_references:
        typeof rawCapability.reference_mode_for_references === 'string'
          ? rawCapability.reference_mode_for_references
          : undefined,
      reference_mode_for_frames:
        typeof rawCapability.reference_mode_for_frames === 'string'
          ? rawCapability.reference_mode_for_frames
          : undefined,
      frames_as_reference_images: parseVideoCapabilityFlag(
        rawCapability.frames_as_reference_images
      ),
      omit_parameters: Array.isArray(rawCapability.omit_parameters)
        ? rawCapability.omit_parameters.map(String)
        : undefined,
      fixed_parameters: isJsonObjectValue(rawCapability.fixed_parameters)
        ? rawCapability.fixed_parameters
        : undefined,
    })
  }
  return capabilities
}

export const channelFormSchema = z
  .object({
    name: z.string().min(1, ERROR_MESSAGES.REQUIRED_NAME),
    type: z.number().min(0, ERROR_MESSAGES.REQUIRED_TYPE),
    base_url: z.string().optional(),
    key: z.string(),
    openai_organization: z.string().optional(),
    models: z.string().min(1, ERROR_MESSAGES.REQUIRED_MODELS),
    group: z.array(z.string()).min(1, ERROR_MESSAGES.REQUIRED_GROUP),
    model_mapping: z
      .string()
      .optional()
      .refine(
        isOptionalModelMapping,
        'Model mapping must be a JSON object with string values'
      ),
    priority: z.number().optional(),
    weight: z.number().optional(),
    test_model: z.string().optional(),
    auto_ban: z.number().optional(),
    status: z.number(),
    status_code_mapping: z
      .string()
      .optional()
      .refine(
        isOptionalStatusCodeMapping,
        'Status code mapping must use valid HTTP status codes'
      ),
    tag: z.string().optional(),
    remark: z
      .string()
      .max(255, 'Remark must be less than 255 characters')
      .optional(),
    setting: z
      .string()
      .optional()
      .refine(isOptionalJsonObject, ERROR_MESSAGES.INVALID_JSON),
    param_override: z
      .string()
      .optional()
      .refine(isOptionalJsonObject, ERROR_MESSAGES.INVALID_JSON),
    header_override: z
      .string()
      .optional()
      .refine(isOptionalJsonObject, ERROR_MESSAGES.INVALID_JSON),
    settings: z
      .string()
      .optional()
      .refine(isOptionalJsonObject, ERROR_MESSAGES.INVALID_JSON),
    advanced_custom: z.string().optional(),
    other: z.string().optional(),
    // Multi-key options (not sent to backend directly)
    multi_key_mode: z.enum(['single', 'batch', 'multi_to_single']).optional(),
    multi_key_type: z.enum(['random', 'polling']).optional(),
    batch_add_set_key_prefix_2_name: z.boolean().optional(),
    key_mode: z.enum(['append', 'replace']).optional(), // For editing multi-key channels
    // Channel extra settings (stored in setting JSON, not sent directly)
    force_format: z.boolean().optional(),
    thinking_to_content: z.boolean().optional(),
    proxy: z.string().optional(),
    video_content_proxy_enabled: z.boolean().optional(),
    video_s3_storage_enabled: z.boolean().optional(),
    video_s3_preferred: z.boolean().optional(),
    video_protocol: z
      .enum([
        '',
        'openai_video',
        'megabyai',
        'globalaiopc',
        'agnes_video_v2',
      ])
      .optional(),
    video_model_capabilities: z.array(videoModelCapabilitySchema).optional(),
    pass_through_body_enabled: z.boolean().optional(),
    system_prompt: z.string().optional(),
    system_prompt_override: z.boolean().optional(),
    // Type-specific settings (stored in settings JSON)
    is_enterprise_account: z.boolean().optional(), // OpenRouter specific
    vertex_key_type: z.enum(['json', 'api_key']).optional(), // Vertex AI specific
    aws_key_type: z.enum(['ak_sk', 'api_key']).optional(), // AWS specific
    azure_responses_version: z.string().optional(), // Azure specific
    // Field passthrough controls (stored in settings JSON)
    allow_service_tier: z.boolean().optional(), // OpenAI/Anthropic
    disable_store: z.boolean().optional(), // OpenAI only
    allow_safety_identifier: z.boolean().optional(), // OpenAI only
    allow_include_obfuscation: z.boolean().optional(), // OpenAI: include usage obfuscation
    allow_inference_geo: z.boolean().optional(), // OpenAI/Anthropic: inference geography
    allow_speed: z.boolean().optional(), // Anthropic: speed mode control
    claude_beta_query: z.boolean().optional(), // Anthropic: beta query passthrough
    disable_task_polling_sleep: z.boolean().optional(),
    // Upstream model update settings (stored in settings JSON)
    upstream_model_update_check_enabled: z.boolean().optional(),
    upstream_model_update_auto_sync_enabled: z.boolean().optional(),
    upstream_model_update_ignored_models: z.string().optional(),
  })
  .superRefine((data, ctx) => {
    if ([3, 8, 36, 45].includes(data.type) && !data.base_url?.trim()) {
      addRequiredIssue(
        ctx,
        'base_url',
        'Base URL is required for this channel type'
      )
    }

    if (data.type === CHANNEL_TYPE_ADVANCED_CUSTOM) {
      const advancedCustomConfig = parseAdvancedCustomConfig(
        data.advanced_custom
      )
      const advancedCustomError =
        validateAdvancedCustomConfig(advancedCustomConfig)
      if (advancedCustomError) {
        addRequiredIssue(ctx, 'advanced_custom', advancedCustomError.message)
      }
      if (
        advancedCustomConfigUsesRelativeUpstreamPath(advancedCustomConfig) &&
        !data.base_url?.trim()
      ) {
        addRequiredIssue(
          ctx,
          'base_url',
          'Base URL is required when an advanced route uses an upstream path'
        )
      }
    }

    if ([3, 18, 21, 39, 41, 49].includes(data.type) && !data.other?.trim()) {
      addRequiredIssue(
        ctx,
        'other',
        'This channel type requires additional configuration'
      )
    }

    if (data.type === 57) {
      if (data.multi_key_mode && data.multi_key_mode !== 'single') {
        addRequiredIssue(
          ctx,
          'multi_key_mode',
          'Codex channels do not support batch creation'
        )
      }
      if (data.key?.trim() && !isCodexCredential(data.key)) {
        addRequiredIssue(
          ctx,
          'key',
          'Codex credential must be a JSON object with access_token and account_id'
        )
      }
    }

    if (
      data.type === 41 &&
      data.vertex_key_type === 'json' &&
      data.key?.trim() &&
      !isVertexJsonKey(data.key)
    ) {
      addRequiredIssue(
        ctx,
        'key',
        'Vertex AI service account key must be valid JSON'
      )
    }

    if (
      data.type === 41 &&
      data.vertex_key_type === 'api_key' &&
      data.multi_key_mode &&
      data.multi_key_mode !== 'single'
    ) {
      addRequiredIssue(
        ctx,
        'multi_key_mode',
        'Vertex AI API Key mode does not support batch creation'
      )
    }

    const configuredModels = new Set<string>()
    if (data.video_protocol && !data.video_model_capabilities?.length) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['video_model_capabilities'],
        message: 'Configure at least one video model',
      })
    }
    for (const [index, capability] of (
      data.video_model_capabilities || []
    ).entries()) {
      const normalizedModel = capability.model.trim().toLowerCase()
      if (!normalizedModel) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['video_model_capabilities', index, 'model'],
          message: 'Model ID is required',
        })
        continue
      }
      if (configuredModels.has(normalizedModel)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['video_model_capabilities', index, 'model'],
          message: 'Model IDs must be unique',
        })
      }
      configuredModels.add(normalizedModel)

      const usesExtendedCapabilities =
        data.video_protocol === 'megabyai' ||
        data.video_protocol === 'globalaiopc'
      if (usesExtendedCapabilities) {
        for (const name of [
          'ratio_required',
          'supports_duration',
          'duration_required',
          'supports_generate_audio',
          'generate_audio_required',
          'supports_first_frame',
          'first_frame_required',
          'supports_last_frame',
          'last_frame_required',
          'last_frame_requires_first_frame',
          'reference_images_incompatible_with_frames',
          'audio_reference_requires_visual_reference',
          'reference_media_incompatible_with_frames',
          'supports_seed',
          'supports_watermark',
        ] as const) {
          if (capability[name] === undefined) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['video_model_capabilities', index, name],
              message: 'This capability setting is required',
            })
          }
        }
        for (const name of [
          'min_reference_images',
          'min_reference_videos',
          'min_reference_audios',
        ] as const) {
          if (capability[name] === undefined) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['video_model_capabilities', index, name],
              message: 'This capability setting is required',
            })
          }
        }
        if (
          capability.supports_duration &&
          capability.min_duration_seconds === undefined
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'min_duration_seconds'],
            message: 'Minimum duration is required',
          })
        }
        if (
          capability.supports_duration &&
          capability.max_duration_seconds === undefined
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'max_duration_seconds'],
            message: 'Maximum duration is required',
          })
        }
        if (
          capability.min_duration_seconds !== undefined &&
          capability.max_duration_seconds !== undefined &&
          capability.min_duration_seconds > capability.max_duration_seconds
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'max_duration_seconds'],
            message: 'Minimum duration cannot exceed maximum duration',
          })
        }
        if (capability.duration_required && !capability.supports_duration) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'duration_required'],
            message: 'Required duration must also be supported',
          })
        }
        if (capability.ratio_required && !capability.ratios?.length) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'ratios'],
            message: 'Configure at least one ratio',
          })
        }
        if (
          capability.generate_audio_required &&
          !capability.supports_generate_audio
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [
              'video_model_capabilities',
              index,
              'generate_audio_required',
            ],
            message: 'Required native audio must also be supported',
          })
        }
        if (
          capability.supports_last_frame &&
          capability.last_frame_requires_first_frame &&
          !capability.supports_first_frame
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [
              'video_model_capabilities',
              index,
              'last_frame_requires_first_frame',
            ],
            message:
              'First frame support is required when the last frame depends on it',
          })
        }
        if (capability.first_frame_required && !capability.supports_first_frame) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'first_frame_required'],
            message: 'Required first frame must also be supported',
          })
        }
        if (capability.last_frame_required && !capability.supports_last_frame) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'last_frame_required'],
            message: 'Required last frame must also be supported',
          })
        }
        if (
          capability.supports_seed &&
          (capability.min_seed === undefined || capability.max_seed === undefined)
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['video_model_capabilities', index, 'min_seed'],
            message: 'Seed range is required when seed is supported',
          })
        }
      }
    }
  })

export type ChannelFormValues = z.infer<typeof channelFormSchema>

// ============================================================================
// Default Form Values
// ============================================================================

export const CHANNEL_FORM_DEFAULT_VALUES: ChannelFormValues = {
  name: '',
  type: 1,
  base_url: '',
  key: '',
  openai_organization: '',
  models: '',
  group: ['default'],
  model_mapping: '',
  priority: 0,
  weight: 0,
  test_model: '',
  auto_ban: 1,
  status: CHANNEL_STATUS.ENABLED,
  status_code_mapping: '',
  tag: '',
  remark: '',
  setting: '',
  param_override: '',
  header_override: '',
  settings: '{}',
  other: '',
  multi_key_mode: 'single',
  multi_key_type: 'random',
  batch_add_set_key_prefix_2_name: false,
  key_mode: 'append',
  // Channel extra settings
  force_format: false,
  thinking_to_content: false,
  proxy: '',
  video_content_proxy_enabled: false,
  video_s3_storage_enabled: false,
  video_s3_preferred: false,
  video_protocol: '',
  video_model_capabilities: [],
  pass_through_body_enabled: false,
  system_prompt: '',
  system_prompt_override: false,
  // Type-specific settings
  is_enterprise_account: false,
  vertex_key_type: 'json',
  aws_key_type: 'ak_sk',
  azure_responses_version: '',
  // Field passthrough controls
  allow_service_tier: false,
  disable_store: false,
  allow_safety_identifier: false,
  allow_include_obfuscation: false,
  allow_inference_geo: false,
  allow_speed: false,
  claude_beta_query: false,
  disable_task_polling_sleep: false,
  upstream_model_update_check_enabled: false,
  upstream_model_update_auto_sync_enabled: false,
  upstream_model_update_ignored_models: '',
  advanced_custom: '',
}

// ============================================================================
// Transform Functions
// ============================================================================

/**
 * Transform Channel from API to Form default values
 */
export function transformChannelToFormDefaults(
  channel: Channel
): ChannelFormValues {
  // Parse channel extra settings from setting field
  let extraSettings = {
    force_format: false,
    thinking_to_content: false,
    proxy: '',
    video_content_proxy_enabled: false,
    video_s3_storage_enabled: false,
    video_s3_preferred: false,
    video_protocol: '' as const,
    video_model_capabilities: [] as VideoModelCapabilityFormValue[],
    pass_through_body_enabled: false,
    system_prompt: '',
    system_prompt_override: false,
  }

  if (channel.setting) {
    try {
      const parsed = JSON.parse(channel.setting)
      const videoProtocol = [
        'openai_video',
        'megabyai',
        'globalaiopc',
        'agnes_video_v2',
      ].includes(parsed.video_protocol)
        ? parsed.video_protocol
        : ''
      extraSettings = {
        force_format: parsed.force_format || false,
        thinking_to_content: parsed.thinking_to_content || false,
        proxy: parsed.proxy || '',
        video_content_proxy_enabled:
          parsed.video_content_proxy_enabled || false,
        video_s3_storage_enabled: parsed.video_s3_storage_enabled || false,
        video_s3_preferred: parsed.video_s3_preferred || false,
        video_protocol: videoProtocol,
        video_model_capabilities: parseVideoModelCapabilities(
          parsed.video_model_capabilities
        ),
        pass_through_body_enabled: parsed.pass_through_body_enabled || false,
        system_prompt: parsed.system_prompt || '',
        system_prompt_override: parsed.system_prompt_override || false,
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to parse channel setting:', error)
    }
  }

  // Parse type-specific settings from settings field
  let vertexKeyType: 'json' | 'api_key' = 'json'
  let azureResponsesVersion = ''
  let isEnterpriseAccount = false
  let awsKeyType: 'ak_sk' | 'api_key' = 'ak_sk'
  let allowServiceTier = false
  let disableStore = false
  let allowSafetyIdentifier = false
  let allowIncludeObfuscation = false
  let allowInferenceGeo = false
  let allowSpeed = false
  let claudeBetaQuery = false
  let disableTaskPollingSleep = false
  let upstreamModelUpdateCheckEnabled = false
  let upstreamModelUpdateAutoSyncEnabled = false
  let upstreamModelUpdateIgnoredModels = ''
  let advancedCustom = ''

  if (channel.settings) {
    try {
      const parsed = JSON.parse(channel.settings)
      vertexKeyType = parsed.vertex_key_type || 'json'
      azureResponsesVersion = parsed.azure_responses_version || ''
      isEnterpriseAccount = parsed.openrouter_enterprise === true
      awsKeyType = parsed.aws_key_type || 'ak_sk'
      allowServiceTier = parsed.allow_service_tier === true
      disableStore = parsed.disable_store === true
      allowSafetyIdentifier = parsed.allow_safety_identifier === true
      allowIncludeObfuscation = parsed.allow_include_obfuscation === true
      allowInferenceGeo = parsed.allow_inference_geo === true
      allowSpeed = parsed.allow_speed === true
      claudeBetaQuery = parsed.claude_beta_query === true
      disableTaskPollingSleep = parsed.disable_task_polling_sleep === true
      upstreamModelUpdateCheckEnabled =
        parsed.upstream_model_update_check_enabled === true
      upstreamModelUpdateAutoSyncEnabled =
        parsed.upstream_model_update_auto_sync_enabled === true
      upstreamModelUpdateIgnoredModels = Array.isArray(
        parsed.upstream_model_update_ignored_models
      )
        ? parsed.upstream_model_update_ignored_models.join(',')
        : ''
      if (parsed.advanced_custom) {
        advancedCustom = stringifyAdvancedCustomConfig(parsed.advanced_custom)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to parse channel settings:', error)
    }
  }

  return {
    name: channel.name || '',
    type: channel.type,
    base_url: channel.base_url || '',
    key: '', // Never populate key from backend for security
    openai_organization: channel.openai_organization || '',
    models: channel.models || '',
    group: parseGroups(channel.group || 'default'),
    model_mapping: channel.model_mapping || '',
    priority: channel.priority || 0,
    weight: channel.weight || 0,
    test_model: channel.test_model || '',
    auto_ban: channel.auto_ban ?? 1,
    status: channel.status,
    status_code_mapping: channel.status_code_mapping || '',
    tag: channel.tag || '',
    remark: channel.remark || '',
    setting: channel.setting || '',
    param_override: channel.param_override || '',
    header_override: channel.header_override || '',
    settings: channel.settings || '{}',
    other: channel.other || '',
    multi_key_mode: 'single',
    multi_key_type: channel.channel_info.multi_key_mode || 'random',
    batch_add_set_key_prefix_2_name: false,
    key_mode: 'append', // Default to append mode for editing multi-key channels
    // Channel extra settings
    ...extraSettings,
    // Type-specific settings
    is_enterprise_account: isEnterpriseAccount,
    vertex_key_type: vertexKeyType,
    azure_responses_version: azureResponsesVersion,
    aws_key_type: awsKeyType,
    allow_service_tier: allowServiceTier,
    disable_store: disableStore,
    allow_include_obfuscation: allowIncludeObfuscation,
    allow_inference_geo: allowInferenceGeo,
    allow_speed: allowSpeed,
    claude_beta_query: claudeBetaQuery,
    disable_task_polling_sleep: disableTaskPollingSleep,
    allow_safety_identifier: allowSafetyIdentifier,
    upstream_model_update_check_enabled: upstreamModelUpdateCheckEnabled,
    upstream_model_update_auto_sync_enabled: upstreamModelUpdateAutoSyncEnabled,
    upstream_model_update_ignored_models: upstreamModelUpdateIgnoredModels,
    advanced_custom: advancedCustom,
  }
}

/**
 * Build the setting JSON string from form extra settings
 */
function buildSettingJSON(formData: ChannelFormValues): string {
  const videoModelCapabilities = Object.fromEntries(
    (formData.video_model_capabilities || [])
      .filter((capability) => capability.model.trim())
      .map((capability) => [
        capability.model.trim(),
        {
          resolutions: capability.resolutions.map((resolution) =>
            resolution.toLowerCase()
          ),
          ratios: (capability.ratios || []).map((ratio) => ratio.toLowerCase()),
          resolution_mappings: capability.resolution_mappings,
          ratio_required: capability.ratio_required,
          min_reference_images: capability.min_reference_images,
          max_reference_images: capability.max_reference_images,
          min_reference_videos: capability.min_reference_videos,
          max_reference_videos: capability.max_reference_videos,
          min_reference_audios: capability.min_reference_audios,
          max_reference_audios: capability.max_reference_audios,
          supports_duration: capability.supports_duration,
          duration_required: capability.duration_required,
          min_duration_seconds: capability.min_duration_seconds,
          max_duration_seconds: capability.max_duration_seconds,
          supports_generate_audio: capability.supports_generate_audio,
          generate_audio_required: capability.generate_audio_required,
          supports_first_frame: capability.supports_first_frame,
          first_frame_required: capability.first_frame_required,
          supports_last_frame: capability.supports_last_frame,
          last_frame_required: capability.last_frame_required,
          last_frame_requires_first_frame:
            capability.last_frame_requires_first_frame,
          reference_images_incompatible_with_frames:
            capability.reference_images_incompatible_with_frames,
          audio_reference_requires_visual_reference:
            capability.audio_reference_requires_visual_reference,
          reference_media_incompatible_with_frames:
            capability.reference_media_incompatible_with_frames,
          supports_seed: capability.supports_seed,
          min_seed: capability.min_seed,
          max_seed: capability.max_seed,
          supports_watermark: capability.supports_watermark,
          auto_reference_mode: capability.auto_reference_mode,
          reference_mode_for_references:
            capability.reference_mode_for_references,
          reference_mode_for_frames: capability.reference_mode_for_frames,
          frames_as_reference_images: capability.frames_as_reference_images,
          omit_parameters: capability.omit_parameters,
          fixed_parameters: capability.fixed_parameters,
        },
      ])
  )
  const settingObj = {
    force_format: formData.force_format || false,
    thinking_to_content: formData.thinking_to_content || false,
    proxy: formData.proxy || '',
    video_content_proxy_enabled: formData.video_content_proxy_enabled || false,
    video_s3_storage_enabled: formData.video_s3_storage_enabled || false,
    video_s3_preferred: formData.video_s3_preferred || false,
    video_protocol: formData.video_protocol || '',
    video_model_capabilities: videoModelCapabilities,
    pass_through_body_enabled: formData.pass_through_body_enabled || false,
    system_prompt: formData.system_prompt || '',
    system_prompt_override: formData.system_prompt_override || false,
  }
  return JSON.stringify(settingObj)
}

/**
 * Build the settings JSON string (for type-specific config like vertex_key_type)
 */
function buildSettingsJSON(formData: ChannelFormValues): string {
  let settingsObj: Record<string, unknown> = {}

  // Try to parse existing settings first
  if (formData.settings && formData.settings !== '{}') {
    try {
      settingsObj = JSON.parse(formData.settings)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to parse existing settings:', error)
    }
  }

  // Add vertex_key_type for Vertex AI channels (type 41)
  if (formData.type === 41) {
    settingsObj.vertex_key_type = formData.vertex_key_type || 'json'
  } else if ('vertex_key_type' in settingsObj) {
    delete settingsObj.vertex_key_type
  }

  // Add azure_responses_version for Azure channels (type 3)
  if (formData.type === 3 && formData.azure_responses_version) {
    settingsObj.azure_responses_version = formData.azure_responses_version
  } else if ('azure_responses_version' in settingsObj) {
    delete settingsObj.azure_responses_version
  }

  // Add enterprise account setting for OpenRouter (type 20)
  if (formData.type === 20) {
    settingsObj.openrouter_enterprise = formData.is_enterprise_account === true
  } else if ('openrouter_enterprise' in settingsObj) {
    delete settingsObj.openrouter_enterprise
  }

  // Add aws_key_type for AWS channels (type 33)
  if (formData.type === 33) {
    settingsObj.aws_key_type = formData.aws_key_type || 'ak_sk'
  } else if ('aws_key_type' in settingsObj) {
    delete settingsObj.aws_key_type
  }

  // Field passthrough controls:
  // - OpenAI (type 1) and Anthropic (type 14): allow_service_tier
  // - OpenAI only: disable_store, allow_safety_identifier
  if (formData.type === 1 || formData.type === 14 || formData.type === 57) {
    settingsObj.allow_service_tier = formData.allow_service_tier === true
  } else if ('allow_service_tier' in settingsObj) {
    delete settingsObj.allow_service_tier
  }

  if (formData.type === 1 || formData.type === 57) {
    settingsObj.disable_store = formData.disable_store === true
    settingsObj.allow_safety_identifier =
      formData.allow_safety_identifier === true
    settingsObj.allow_include_obfuscation =
      formData.allow_include_obfuscation === true
    settingsObj.allow_inference_geo = formData.allow_inference_geo === true
  } else {
    if ('disable_store' in settingsObj) {
      delete settingsObj.disable_store
    }
    if ('allow_safety_identifier' in settingsObj) {
      delete settingsObj.allow_safety_identifier
    }
    if ('allow_include_obfuscation' in settingsObj) {
      delete settingsObj.allow_include_obfuscation
    }
    if (formData.type !== 14 && 'allow_inference_geo' in settingsObj) {
      delete settingsObj.allow_inference_geo
    }
  }

  // Anthropic (type 14): claude_beta_query, allow_inference_geo, allow_speed
  if (formData.type === 14) {
    settingsObj.allow_inference_geo = formData.allow_inference_geo === true
    settingsObj.allow_speed = formData.allow_speed === true
    settingsObj.claude_beta_query = formData.claude_beta_query === true
  } else {
    if ('allow_speed' in settingsObj) delete settingsObj.allow_speed
    if ('claude_beta_query' in settingsObj) delete settingsObj.claude_beta_query
  }

  settingsObj.disable_task_polling_sleep =
    formData.disable_task_polling_sleep === true

  // Upstream model update settings (for model-fetchable channel types)
  if (MODEL_FETCHABLE_TYPES.has(formData.type)) {
    settingsObj.upstream_model_update_check_enabled =
      formData.upstream_model_update_check_enabled === true
    settingsObj.upstream_model_update_auto_sync_enabled =
      settingsObj.upstream_model_update_check_enabled === true &&
      formData.upstream_model_update_auto_sync_enabled === true
    settingsObj.upstream_model_update_ignored_models = [
      ...new Set(
        String(formData.upstream_model_update_ignored_models || '')
          .split(',')
          .map((model) => model.trim())
          .filter(Boolean)
      ),
    ]
    if (
      !Array.isArray(settingsObj.upstream_model_update_last_detected_models) ||
      settingsObj.upstream_model_update_check_enabled !== true
    ) {
      settingsObj.upstream_model_update_last_detected_models = []
    }
    if (typeof settingsObj.upstream_model_update_last_check_time !== 'number') {
      settingsObj.upstream_model_update_last_check_time = 0
    }
  }

  if (formData.type === CHANNEL_TYPE_ADVANCED_CUSTOM) {
    const advancedCustomConfig = parseAdvancedCustomConfig(
      formData.advanced_custom
    )
    if (advancedCustomConfig) {
      settingsObj.advanced_custom = advancedCustomConfig
    }
  } else if ('advanced_custom' in settingsObj) {
    delete settingsObj.advanced_custom
  }

  return JSON.stringify(settingsObj)
}

function normalizeBaseUrl(value: string | undefined): string {
  return String(value || '')
    .trim()
    .replace(/\/+$/, '')
}

/**
 * Transform form data to API payload for creating channel
 */
export function transformFormDataToCreatePayload(formData: ChannelFormValues): {
  mode: 'single' | 'batch' | 'multi_to_single'
  multi_key_mode?: 'random' | 'polling'
  batch_add_set_key_prefix_2_name?: boolean
  channel: Partial<Channel>
} {
  const mode = formData.multi_key_mode || 'single'

  const channel: Partial<Channel> = {
    name: formData.name,
    type: formData.type,
    base_url: normalizeBaseUrl(formData.base_url) || null,
    key: formData.key,
    openai_organization: formData.openai_organization || null,
    models: formData.models,
    group: formatGroups(formData.group),
    model_mapping: formData.model_mapping || null,
    priority: formData.priority || null,
    weight: formData.weight || null,
    test_model: formData.test_model || null,
    auto_ban: formData.auto_ban ?? 1,
    status: formData.status,
    status_code_mapping: formData.status_code_mapping || null,
    tag: formData.tag || null,
    remark: formData.remark || '',
    setting: buildSettingJSON(formData),
    param_override: formData.param_override || null,
    header_override: formData.header_override || null,
    settings: buildSettingsJSON(formData),
    other: formData.other || '',
  }

  // Clean up empty strings to null for optional fields
  Object.keys(channel).forEach((key) => {
    if (channel[key as keyof typeof channel] === '') {
      ;(channel as Record<string, unknown>)[key] = null
    }
  })

  return {
    mode,
    multi_key_mode:
      mode === 'multi_to_single' ? formData.multi_key_type : undefined,
    batch_add_set_key_prefix_2_name:
      mode === 'batch' ? formData.batch_add_set_key_prefix_2_name : undefined,
    channel,
  }
}

/**
 * Transform form data to API payload for updating channel
 */
export function transformFormDataToUpdatePayload(
  formData: ChannelFormValues,
  channelId: number
): Partial<Channel> {
  const payload: Partial<Channel> = {
    id: channelId,
    name: formData.name,
    type: formData.type,
    base_url: normalizeBaseUrl(formData.base_url) || null,
    openai_organization: formData.openai_organization || null,
    models: formData.models,
    group: formatGroups(formData.group),
    model_mapping: formData.model_mapping || null,
    priority: formData.priority ?? 0,
    weight: formData.weight ?? 0,
    test_model: formData.test_model || null,
    auto_ban: formData.auto_ban ?? 1,
    status_code_mapping: formData.status_code_mapping || null,
    tag: formData.tag || null,
    remark: formData.remark || '',
    setting: buildSettingJSON(formData),
    param_override: formData.param_override || null,
    header_override: formData.header_override || null,
    settings: buildSettingsJSON(formData),
    other: formData.other || '',
  }

  // Only include key if it was changed (not empty)
  if (formData.key && formData.key.trim()) {
    payload.key = formData.key
  }

  // Clean up empty strings to null for optional fields
  Object.keys(payload).forEach((key) => {
    if (payload[key as keyof typeof payload] === '') {
      ;(payload as Record<string, unknown>)[key] = null
    }
  })

  // Send explicit empty strings for nullable fields so GORM updates can clear them.
  payload.base_url = normalizeBaseUrl(formData.base_url) || ''
  payload.openai_organization = formData.openai_organization || ''
  payload.test_model = formData.test_model || ''
  payload.tag = formData.tag || ''
  payload.remark = formData.remark || ''
  payload.model_mapping = formData.model_mapping || ''
  payload.status_code_mapping = formData.status_code_mapping || ''
  payload.param_override = formData.param_override || ''
  payload.header_override = formData.header_override || ''

  return payload
}

// ============================================================================
// Validation Helpers
// ============================================================================

/**
 * Validate JSON string
 */
export function validateJSON(value: string): boolean {
  if (!value || value.trim() === '') return true
  try {
    JSON.parse(value)
    return true
  } catch {
    return false
  }
}

/**
 * Validate model mapping format
 */
export function validateModelMapping(value: string): boolean {
  if (!value || value.trim() === '') return true
  return validateJSON(value)
}

/**
 * Parse models string to array
 */
export function parseModels(models: string): string[] {
  if (!models) return []
  return models
    .split(',')
    .map((m) => m.trim())
    .filter((m) => m.length > 0)
}

/**
 * Parse groups string to array
 */
export function parseGroups(groups: string): string[] {
  if (!groups) return []
  return groups
    .split(',')
    .map((g) => g.trim())
    .filter((g) => g.length > 0)
}

/**
 * Format models array to string
 */
export function formatModels(models: string[]): string {
  return models.join(',')
}

/**
 * Format groups array to string
 */
export function formatGroups(groups: string[]): string {
  return groups.join(',')
}
