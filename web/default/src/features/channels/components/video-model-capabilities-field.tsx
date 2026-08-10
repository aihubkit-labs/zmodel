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
import { Plus, Trash2 } from 'lucide-react'
import {
  type Control,
  useFieldArray,
  useFormContext,
  useWatch,
} from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { TagInput } from '@/components/tag-input'
import { Button } from '@/components/ui/button'
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

import {
  MAX_VIDEO_DURATION_SECONDS,
  MAX_VIDEO_REFERENCE_COUNT,
  type ChannelFormValues,
} from '../lib'

type VideoModelCapabilitiesFieldProps = {
  control: Control<ChannelFormValues>
  models: string[]
  protocol: ChannelFormValues['video_protocol']
}

export function VideoModelCapabilitiesField(
  props: VideoModelCapabilitiesFieldProps
) {
  const { t } = useTranslation()
  const { fields, append, remove } = useFieldArray({
    control: props.control,
    name: 'video_model_capabilities',
  })
  const form = useFormContext<ChannelFormValues>()
  const capabilityValues = useWatch({
    control: props.control,
    name: 'video_model_capabilities',
  })
  const isMinimaxH3 = props.protocol === 'minimax-h3(megabyai)'
  const requiresDuration =
    props.protocol === 'seedance(megabyai)' || isMinimaxH3

  return (
    <FormItem className='gap-3 px-4 py-3'>
      <div>
        <FormLabel>{t('Video model capabilities')}</FormLabel>
        <FormDescription>
          {t(
            'Configure each upstream video model and its reference media limits'
          )}
        </FormDescription>
      </div>

      {fields.length === 0 ? (
        <div className='text-muted-foreground flex min-h-16 items-center justify-center rounded-md border border-dashed px-3 text-center text-sm'>
          {t('No video model capabilities configured')}
        </div>
      ) : (
        <div className='space-y-3'>
          {fields.map((capability, index) => (
            <div
              key={capability.id}
              className='space-y-3 rounded-md border p-3'
            >
              <div className='flex items-start gap-2'>
                <FormField
                  control={props.control}
                  name={`video_model_capabilities.${index}.model`}
                  render={({ field }) => (
                    <FormItem className='min-w-0 flex-1'>
                      <FormLabel>{t('Upstream model ID')}</FormLabel>
                      <Select
                        items={props.models.map((model) => ({
                          value: model,
                          label: model,
                        }))}
                        value={field.value || null}
                        onValueChange={field.onChange}
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
              </div>

              <FormField
                control={props.control}
                name={`video_model_capabilities.${index}.resolutions`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Supported resolutions')}</FormLabel>
                    <FormControl>
                      <TagInput
                        value={field.value || []}
                        onChange={field.onChange}
                        placeholder={t('Add a resolution, for example 720p')}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Press Enter or comma to add resolutions')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {isMinimaxH3 ? (
                <FormField
                  control={props.control}
                  name={`video_model_capabilities.${index}.ratios`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Supported aspect ratios')}</FormLabel>
                      <FormControl>
                        <TagInput
                          value={field.value || []}
                          onChange={field.onChange}
                          placeholder={t(
                            'Add an aspect ratio, for example 16:9'
                          )}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Press Enter or comma to add aspect ratios')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ) : null}

              {requiresDuration ? (
                <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                  {(
                    [
                      ['min_duration_seconds', 'Minimum duration (seconds)'],
                      ['max_duration_seconds', 'Maximum duration (seconds)'],
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
                            type='number'
                            min={0}
                            max={MAX_VIDEO_REFERENCE_COUNT}
                            step={1}
                            value={field.value}
                            onChange={(event) =>
                              field.onChange(
                                event.target.value === ''
                                  ? 0
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

              {isMinimaxH3 ? (
                <div className='grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2'>
                  {(
                    [
                      ['supports_generate_audio', 'Supports native audio'],
                      [
                        'generate_audio_required',
                        'Require generate_audio=true',
                      ],
                      ['supports_first_frame', 'Supports first frame'],
                      ['supports_last_frame', 'Supports last frame'],
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
                        'Reference audio requires a reference image',
                      ],
                    ] as const
                  ).map(([name, label]) => (
                    <FormField
                      key={name}
                      control={props.control}
                      name={`video_model_capabilities.${index}.${name}`}
                      render={({ field }) => {
                        const capability = capabilityValues?.[index]
                        const disabled =
                          (name === 'generate_audio_required' &&
                            capability?.supports_generate_audio !== true) ||
                          (name === 'last_frame_requires_first_frame' &&
                            (capability?.supports_first_frame !== true ||
                              capability?.supports_last_frame !== true))

                        return (
                          <FormItem
                            className='gap-1'
                            data-disabled={disabled || undefined}
                          >
                            <div className='flex min-h-9 items-center justify-between gap-3'>
                              <FormLabel className='font-normal'>
                                {t(label)}
                              </FormLabel>
                              <FormControl>
                                <Switch
                                  checked={field.value ?? false}
                                  disabled={disabled}
                                  onCheckedChange={(checked) => {
                                    field.onChange(checked)
                                    if (checked) {
                                      return
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
                                  }}
                                />
                              </FormControl>
                            </div>
                            <FormMessage />
                          </FormItem>
                        )
                      }}
                    />
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </div>
      )}

      <Button
        type='button'
        variant='outline'
        size='sm'
        className='w-full'
        disabled={props.models.length === 0}
        onClick={() =>
          append({
            model: '',
            resolutions: [],
            max_reference_images: 0,
            max_reference_videos: 0,
            max_reference_audios: 0,
            ...(isMinimaxH3
              ? {
                  ratios: [],
                  supports_generate_audio: true,
                  generate_audio_required: true,
                  supports_first_frame: true,
                  supports_last_frame: true,
                  last_frame_requires_first_frame: true,
                  reference_images_incompatible_with_frames: true,
                  audio_reference_requires_visual_reference: true,
                }
              : {}),
          })
        }
      >
        <Plus className='size-4' aria-hidden='true' />
        {t('Add model capability')}
      </Button>
      {props.models.length === 0 ? (
        <FormDescription>
          {t('Fetch or add channel models before configuring video models')}
        </FormDescription>
      ) : null}
    </FormItem>
  )
}
