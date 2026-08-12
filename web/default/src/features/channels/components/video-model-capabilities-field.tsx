/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BookOpen,
  ChevronDown,
  Copy,
  ExternalLink,
  Plus,
  Save,
  Trash2,
  WandSparkles,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import {
  type Control,
  useFieldArray,
  useFormContext,
  useWatch,
} from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { TagInput } from '@/components/tag-input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  getVideoModelCapabilityTemplates,
  saveVideoModelCapabilityTemplate,
} from '../api'
import {
  MAX_VIDEO_DURATION_SECONDS,
  MAX_VIDEO_REFERENCE_COUNT,
  type ChannelFormValues,
} from '../lib'
import {
  defaultVideoModelCapability,
  videoModelCapabilityFromTemplate,
} from '../lib/video-model-capability'
import type {
  VideoModelCapability,
  VideoModelCapabilityTemplate,
} from '../types'

type VideoModelCapabilitiesFieldProps = {
  control: Control<ChannelFormValues>
  models: string[]
  protocol: ChannelFormValues['video_protocol']
  templateToolsEnabled?: boolean
  readOnly?: boolean
}

const EXTENDED_PROTOCOLS = new Set(['megabyai', 'globalaiopc'])
const REFERENCE_MEDIA_LABELS = {
  images: 'Reference images',
  videos: 'Reference videos',
  audios: 'Reference audios',
} as const

function JsonObjectField(props: {
  value: Record<string, unknown> | undefined
  onChange: (value: Record<string, unknown>) => void
  placeholder: string
  objectRequiredMessage: string
  invalidJsonMessage: string
  disabled?: boolean
}) {
  const [text, setText] = useState(() =>
    JSON.stringify(props.value || {}, null, 2)
  )
  const [error, setError] = useState('')

  useEffect(() => {
    setText(JSON.stringify(props.value || {}, null, 2))
  }, [props.value])

  return (
    <div className='space-y-1'>
      <Textarea
        className='min-h-24 font-mono text-xs'
        disabled={props.disabled}
        value={text}
        placeholder={props.placeholder}
        onChange={(event) => setText(event.target.value)}
        onBlur={() => {
          try {
            const parsed: unknown = JSON.parse(text || '{}')
            if (
              !parsed ||
              typeof parsed !== 'object' ||
              Array.isArray(parsed)
            ) {
              setError(props.objectRequiredMessage)
              return
            }
            setError('')
            props.onChange(parsed as Record<string, unknown>)
          } catch {
            setError(props.invalidJsonMessage)
          }
        }}
      />
      {error ? <p className='text-destructive text-xs'>{error}</p> : null}
    </div>
  )
}

export function VideoModelCapabilitiesField(
  props: VideoModelCapabilitiesFieldProps
) {
  const { t } = useTranslation()
  const form = useFormContext<ChannelFormValues>()
  const queryClient = useQueryClient()
  const { fields, append, remove, replace, update } = useFieldArray({
    control: props.control,
    name: 'video_model_capabilities',
  })
  const capabilities =
    useWatch({
      control: props.control,
      name: 'video_model_capabilities',
    }) || []
  const extended = EXTENDED_PROTOCOLS.has(props.protocol || '')
  const templateToolsEnabled = props.templateToolsEnabled !== false

  const templatesQuery = useQuery({
    queryKey: ['video-capability-templates', props.protocol],
    queryFn: () => getVideoModelCapabilityTemplates(props.protocol || ''),
    enabled: extended && templateToolsEnabled,
  })
  const templates = useMemo(
    () => templatesQuery.data?.data || [],
    [templatesQuery.data]
  )
  const templatesByModel = useMemo(() => {
    const result = new Map<string, VideoModelCapabilityTemplate>()
    for (const template of templates) {
      result.set(template.model_id.trim().toLowerCase(), template)
    }
    return result
  }, [templates])

  const saveTemplateMutation = useMutation({
    mutationFn: saveVideoModelCapabilityTemplate,
    onSuccess: async (response) => {
      if (!response.success) throw new Error(response.message || 'Save failed')
      await queryClient.invalidateQueries({
        queryKey: ['video-capability-templates', props.protocol],
      })
      toast.success(t('Video capability template saved'))
    },
    onError: (error) => toast.error(error.message),
  })

  const applyTemplate = (
    index: number,
    template: VideoModelCapabilityTemplate,
    selectedModel?: string
  ) => {
    const model =
      selectedModel || capabilities[index]?.model || template.model_id
    update(index, videoModelCapabilityFromTemplate(template, model))
  }

  return (
    <FormItem className='gap-3 px-4 py-3'>
      <div>
        <FormLabel>{t('Video model capabilities')}</FormLabel>
        <FormDescription>
          {t('Select an upstream model and apply its capability template')}
        </FormDescription>
      </div>

      {fields.length === 0 ? (
        <div className='text-muted-foreground flex min-h-16 items-center justify-center rounded-md border border-dashed px-3 text-center text-sm'>
          {t('No video model capabilities configured')}
        </div>
      ) : (
        <div className='space-y-3'>
          {fields.map((fieldItem, index) => {
            const capability =
              capabilities[index] || defaultVideoModelCapability()
            const matchedTemplate = templatesByModel.get(
              capability.model?.trim().toLowerCase() || ''
            )
            const supportsDuration = capability.supports_duration !== false
            const supportsFirstFrame = capability.supports_first_frame === true
            const supportsLastFrame = capability.supports_last_frame === true
            const supportsSeed = capability.supports_seed === true

            return (
              <div
                key={fieldItem.id}
                className='space-y-4 rounded-md border p-3'
              >
                <div className='flex items-start gap-2'>
                  <FormField
                    control={props.control}
                    name={`video_model_capabilities.${index}.model`}
                    render={({ field }) => (
                      <FormItem className='min-w-0 flex-1'>
                        <FormLabel>{t('Upstream model ID')}</FormLabel>
                        <Select
                          disabled={props.readOnly}
                          items={props.models.map((model) => ({
                            value: model,
                            label: model,
                          }))}
                          value={field.value || null}
                          onValueChange={(model) => {
                            if (!model) return
                            field.onChange(model)
                            const template = templatesByModel.get(
                              model.trim().toLowerCase()
                            )
                            if (template) applyTemplate(index, template, model)
                          }}
                        >
                          <FormControl>
                            <SelectTrigger className='w-full'>
                              <SelectValue
                                placeholder={t('Select upstream model ID')}
                              />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent alignItemWithTrigger={false}>
                            <SelectGroup>
                              {props.models.map((model) => (
                                <SelectItem key={model} value={model}>
                                  {model}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  {!props.readOnly ? (
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      className='mt-7 shrink-0'
                      onClick={() => remove(index)}
                      aria-label={t('Remove model capability')}
                      title={t('Remove model capability')}
                    >
                      <Trash2 className='size-4' aria-hidden='true' />
                    </Button>
                  ) : null}
                </div>

                {extended && templateToolsEnabled ? (
                  <div className='bg-muted/40 space-y-2 rounded-md border px-3 py-2'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <BookOpen className='text-muted-foreground size-4' />
                      <span className='text-sm font-medium'>
                        {t('Capability template')}
                      </span>
                      {matchedTemplate ? (
                        <>
                          <Badge variant='secondary'>
                            {matchedTemplate.name}
                          </Badge>
                          <Badge variant='outline'>
                            {matchedTemplate.built_in
                              ? t('Official document preset')
                              : t('Custom template')}
                          </Badge>
                        </>
                      ) : (
                        <span className='text-muted-foreground text-sm'>
                          {t('No exact template match')}
                        </span>
                      )}
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      <Select
                        items={templates.map((template) => ({
                          value: String(template.id),
                          label: `${template.name} (${template.model_id})`,
                        }))}
                        onValueChange={(templateID) => {
                          const template = templates.find(
                            (item) => String(item.id) === templateID
                          )
                          if (template) applyTemplate(index, template)
                        }}
                        value={
                          matchedTemplate ? String(matchedTemplate.id) : null
                        }
                      >
                        <SelectTrigger className='min-w-56 flex-1'>
                          <Copy className='size-4' />
                          <SelectValue
                            placeholder={t('Apply a capability template')}
                          />
                        </SelectTrigger>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            {templates.map((template) => (
                              <SelectItem
                                key={template.id}
                                value={String(template.id)}
                              >
                                {template.name} ({template.model_id})
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      {matchedTemplate?.source_url ? (
                        <Button
                          variant='outline'
                          size='sm'
                          render={
                            <a
                              href={matchedTemplate.source_url}
                              target='_blank'
                              rel='noreferrer'
                            />
                          }
                        >
                          <ExternalLink className='size-4' />
                          {t('Source document')}
                        </Button>
                      ) : null}
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        disabled={
                          !capability.model || saveTemplateMutation.isPending
                        }
                        onClick={() => {
                          const { model, ...templateCapability } = capability
                          saveTemplateMutation.mutate({
                            id: matchedTemplate?.built_in
                              ? 0
                              : matchedTemplate?.id || 0,
                            video_protocol: props.protocol || '',
                            model_id: model,
                            name: model,
                            capability:
                              templateCapability as VideoModelCapability,
                            source: 'manual',
                            source_url: matchedTemplate?.source_url,
                            built_in: false,
                          })
                        }}
                      >
                        <Save className='size-4' />
                        {t('Save current capability as template')}
                      </Button>
                    </div>
                  </div>
                ) : null}

                <div className='grid grid-cols-1 gap-3 lg:grid-cols-2'>
                  <FormField
                    control={props.control}
                    name={`video_model_capabilities.${index}.resolutions`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Supported resolutions')}</FormLabel>
                        <FormControl>
                          <TagInput
                            disabled={props.readOnly}
                            value={field.value || []}
                            onChange={field.onChange}
                            placeholder={t(
                              'Add a resolution, for example 720p'
                            )}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  {extended ? (
                    <FormField
                      control={props.control}
                      name={`video_model_capabilities.${index}.ratios`}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Supported aspect ratios')}</FormLabel>
                          <FormControl>
                            <TagInput
                              disabled={props.readOnly}
                              value={field.value || []}
                              onChange={field.onChange}
                              placeholder={t(
                                'Add an aspect ratio, for example 16:9'
                              )}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  ) : null}
                </div>

                {extended ? (
                  <>
                    <div className='grid grid-cols-1 gap-3 sm:grid-cols-3'>
                      {(['images', 'videos', 'audios'] as const).map(
                        (media) => (
                          <div key={media} className='space-y-2'>
                            <FormLabel>
                              {t(REFERENCE_MEDIA_LABELS[media])}
                            </FormLabel>
                            <div className='grid grid-cols-2 gap-2'>
                              {(['min', 'max'] as const).map((limit) => (
                                <FormField
                                  key={limit}
                                  control={props.control}
                                  name={`video_model_capabilities.${index}.${limit}_reference_${media}`}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel className='text-muted-foreground text-xs font-normal'>
                                        {t(
                                          limit === 'min'
                                            ? 'Minimum'
                                            : 'Maximum'
                                        )}
                                      </FormLabel>
                                      <FormControl>
                                        <Input
                                          disabled={props.readOnly}
                                          type='number'
                                          min={0}
                                          max={MAX_VIDEO_REFERENCE_COUNT}
                                          step={1}
                                          value={field.value ?? 0}
                                          onChange={(event) =>
                                            field.onChange(
                                              event.target.valueAsNumber
                                            )
                                          }
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              ))}
                            </div>
                          </div>
                        )
                      )}
                    </div>

                    <div className='grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2 lg:grid-cols-3'>
                      {(
                        [
                          ['ratio_required', 'Aspect ratio is required'],
                          ['supports_duration', 'Supports duration'],
                          ['duration_required', 'Duration is required'],
                          ['supports_generate_audio', 'Supports native audio'],
                          [
                            'generate_audio_required',
                            'Require generate_audio=true',
                          ],
                          ['supports_first_frame', 'Supports first frame'],
                          ['first_frame_required', 'First frame is required'],
                          ['supports_last_frame', 'Supports last frame'],
                          ['last_frame_required', 'Last frame is required'],
                          [
                            'last_frame_requires_first_frame',
                            'Last frame requires first frame',
                          ],
                          [
                            'reference_images_incompatible_with_frames',
                            'Reference images cannot be combined with frames',
                          ],
                          [
                            'audio_reference_requires_visual_reference',
                            'Reference audio requires a visual reference',
                          ],
                          [
                            'reference_media_incompatible_with_frames',
                            'Reference media cannot be combined with frames',
                          ],
                          ['supports_seed', 'Supports seed'],
                          ['supports_watermark', 'Supports watermark'],
                        ] as const
                      ).map(([name, label]) => {
                        const disabled =
                          (name === 'duration_required' && !supportsDuration) ||
                          (name === 'generate_audio_required' &&
                            capability.supports_generate_audio !== true) ||
                          (name === 'first_frame_required' &&
                            !supportsFirstFrame) ||
                          (name === 'last_frame_required' &&
                            !supportsLastFrame) ||
                          (name === 'last_frame_requires_first_frame' &&
                            (!supportsFirstFrame || !supportsLastFrame))
                        return (
                          <FormField
                            key={name}
                            control={props.control}
                            name={`video_model_capabilities.${index}.${name}`}
                            render={({ field }) => (
                              <FormItem className='flex min-h-9 items-center justify-between gap-3'>
                                <FormLabel className='font-normal'>
                                  {t(label)}
                                </FormLabel>
                                <FormControl>
                                  <Switch
                                    checked={field.value ?? false}
                                    disabled={props.readOnly || disabled}
                                    onCheckedChange={(checked) => {
                                      field.onChange(checked)
                                      if (checked) return
                                      if (name === 'supports_duration') {
                                        form.setValue(
                                          `video_model_capabilities.${index}.duration_required`,
                                          false,
                                          {
                                            shouldDirty: true,
                                            shouldValidate: true,
                                          }
                                        )
                                      }
                                      if (name === 'supports_generate_audio') {
                                        form.setValue(
                                          `video_model_capabilities.${index}.generate_audio_required`,
                                          false,
                                          {
                                            shouldDirty: true,
                                            shouldValidate: true,
                                          }
                                        )
                                      }
                                      if (name === 'supports_first_frame') {
                                        form.setValue(
                                          `video_model_capabilities.${index}.first_frame_required`,
                                          false,
                                          {
                                            shouldDirty: true,
                                            shouldValidate: true,
                                          }
                                        )
                                      }
                                      if (
                                        name === 'supports_first_frame' ||
                                        name === 'supports_last_frame'
                                      ) {
                                        form.setValue(
                                          `video_model_capabilities.${index}.last_frame_requires_first_frame`,
                                          false,
                                          {
                                            shouldDirty: true,
                                            shouldValidate: true,
                                          }
                                        )
                                      }
                                      if (name === 'supports_last_frame') {
                                        form.setValue(
                                          `video_model_capabilities.${index}.last_frame_required`,
                                          false,
                                          {
                                            shouldDirty: true,
                                            shouldValidate: true,
                                          }
                                        )
                                      }
                                    }}
                                  />
                                </FormControl>
                              </FormItem>
                            )}
                          />
                        )
                      })}
                    </div>

                    {supportsDuration ? (
                      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                        {(
                          [
                            [
                              'min_duration_seconds',
                              'Minimum duration (seconds)',
                            ],
                            [
                              'max_duration_seconds',
                              'Maximum duration (seconds)',
                            ],
                          ] as const
                        ).map(([name, label]) => (
                          <FormField
                            key={name}
                            control={props.control}
                            name={`video_model_capabilities.${index}.${name}`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>{t(label)}</FormLabel>
                                <FormControl>
                                  <Input
                                    disabled={props.readOnly}
                                    type='number'
                                    min={1}
                                    max={MAX_VIDEO_DURATION_SECONDS}
                                    step={1}
                                    value={field.value ?? ''}
                                    onChange={(event) =>
                                      field.onChange(
                                        event.target.value === ''
                                          ? undefined
                                          : event.target.valueAsNumber
                                      )
                                    }
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        ))}
                      </div>
                    ) : null}

                    {supportsSeed ? (
                      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                        {(
                          [
                            ['min_seed', 'Minimum seed'],
                            ['max_seed', 'Maximum seed'],
                          ] as const
                        ).map(([name, label]) => (
                          <FormField
                            key={name}
                            control={props.control}
                            name={`video_model_capabilities.${index}.${name}`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>{t(label)}</FormLabel>
                                <FormControl>
                                  <Input
                                    disabled={props.readOnly}
                                    type='number'
                                    min={-1}
                                    max={2147483647}
                                    step={1}
                                    value={field.value ?? ''}
                                    onChange={(event) =>
                                      field.onChange(event.target.valueAsNumber)
                                    }
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        ))}
                      </div>
                    ) : null}

                    <Collapsible>
                      <CollapsibleTrigger
                        render={
                          <Button type='button' variant='ghost' size='sm' />
                        }
                      >
                        <ChevronDown className='size-4' />
                        {t('Advanced upstream mapping')}
                      </CollapsibleTrigger>
                      <CollapsibleContent className='space-y-4 pt-3'>
                        <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                          {(capability.resolutions || []).map((resolution) => (
                            <FormItem key={resolution}>
                              <FormLabel>
                                {t('Upstream value for resolution')}{' '}
                                {resolution}
                              </FormLabel>
                              <Input
                                disabled={props.readOnly}
                                value={
                                  capability.resolution_mappings?.[
                                    resolution
                                  ] || ''
                                }
                                placeholder={resolution}
                                onChange={(event) => {
                                  const mappings = {
                                    ...capability.resolution_mappings,
                                  }
                                  if (event.target.value.trim()) {
                                    mappings[resolution] = event.target.value
                                  } else {
                                    delete mappings[resolution]
                                  }
                                  form.setValue(
                                    `video_model_capabilities.${index}.resolution_mappings`,
                                    mappings,
                                    { shouldDirty: true }
                                  )
                                }}
                              />
                            </FormItem>
                          ))}
                        </div>
                        <div className='grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2'>
                          {(
                            [
                              [
                                'auto_reference_mode',
                                'Automatically set reference_mode',
                              ],
                              [
                                'frames_as_reference_images',
                                'Send frames as reference images',
                              ],
                            ] as const
                          ).map(([name, label]) => (
                            <FormField
                              key={name}
                              control={props.control}
                              name={`video_model_capabilities.${index}.${name}`}
                              render={({ field }) => (
                                <FormItem className='flex min-h-9 items-center justify-between gap-3'>
                                  <FormLabel className='font-normal'>
                                    {t(label)}
                                  </FormLabel>
                                  <FormControl>
                                    <Switch
                                      checked={field.value ?? false}
                                      disabled={props.readOnly}
                                      onCheckedChange={field.onChange}
                                    />
                                  </FormControl>
                                </FormItem>
                              )}
                            />
                          ))}
                        </div>
                        {capability.auto_reference_mode ? (
                          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                            {(
                              [
                                [
                                  'reference_mode_for_references',
                                  'Reference mode for media',
                                ],
                                [
                                  'reference_mode_for_frames',
                                  'Reference mode for frames',
                                ],
                              ] as const
                            ).map(([name, label]) => (
                              <FormField
                                key={name}
                                control={props.control}
                                name={`video_model_capabilities.${index}.${name}`}
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>{t(label)}</FormLabel>
                                    <FormControl>
                                      <Input
                                        disabled={props.readOnly}
                                        value={field.value || ''}
                                        onChange={field.onChange}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            ))}
                          </div>
                        ) : null}
                        <FormField
                          control={props.control}
                          name={`video_model_capabilities.${index}.omit_parameters`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t('Parameters omitted upstream')}
                              </FormLabel>
                              <FormControl>
                                <TagInput
                                  disabled={props.readOnly}
                                  value={field.value || []}
                                  onChange={field.onChange}
                                  placeholder={t('Add a parameter name')}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={props.control}
                          name={`video_model_capabilities.${index}.fixed_parameters`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t('Fixed upstream parameters')}
                              </FormLabel>
                              <FormControl>
                                <JsonObjectField
                                  value={field.value}
                                  onChange={field.onChange}
                                  placeholder='{"face_processing": true}'
                                  objectRequiredMessage={t('Invalid JSON')}
                                  invalidJsonMessage={t('Invalid JSON')}
                                  disabled={props.readOnly}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </CollapsibleContent>
                    </Collapsible>
                  </>
                ) : (
                  <div className='grid grid-cols-1 gap-3 sm:grid-cols-3'>
                    {(
                      [
                        ['max_reference_images', 'Max reference images'],
                        ['max_reference_videos', 'Max reference videos'],
                        ['max_reference_audios', 'Max reference audios'],
                      ] as const
                    ).map(([name, label]) => (
                      <FormField
                        key={name}
                        control={props.control}
                        name={`video_model_capabilities.${index}.${name}`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t(label)}</FormLabel>
                            <FormControl>
                              <Input
                                disabled={props.readOnly}
                                type='number'
                                min={0}
                                max={MAX_VIDEO_REFERENCE_COUNT}
                                value={field.value}
                                onChange={(event) =>
                                  field.onChange(event.target.valueAsNumber)
                                }
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {!props.readOnly ? (
        <Button
          type='button'
          variant='outline'
          size='sm'
          className='w-full'
          disabled={props.models.length === 0}
          onClick={() => append(defaultVideoModelCapability())}
        >
          <Plus className='size-4' aria-hidden='true' />
          {t('Add model capability')}
        </Button>
      ) : null}
      {!props.readOnly &&
      extended &&
      templates.length > 0 &&
      fields.length === 0 ? (
        <Button
          type='button'
          variant='secondary'
          size='sm'
          className='w-full'
          onClick={() => {
            const availableModels = new Set(
              props.models.map((model) => model.toLowerCase())
            )
            replace(
              templates
                .filter((template) =>
                  availableModels.has(template.model_id.toLowerCase())
                )
                .map((template) =>
                  videoModelCapabilityFromTemplate(template, template.model_id)
                )
            )
          }}
        >
          <WandSparkles className='size-4' />
          {t('Apply templates for all matching channel models')}
        </Button>
      ) : null}
      {props.models.length === 0 ? (
        <FormDescription>
          {t('Fetch or add channel models before configuring video models')}
        </FormDescription>
      ) : null}
    </FormItem>
  )
}
