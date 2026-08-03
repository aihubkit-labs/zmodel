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
import { type Control, useFieldArray } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

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
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import {
  VIDEO_RESOLUTIONS,
  type ChannelFormValues,
  type VideoResolution,
} from '../lib'

type SeedanceModelCapabilitiesFieldProps = {
  control: Control<ChannelFormValues>
}

export function SeedanceModelCapabilitiesField(
  props: SeedanceModelCapabilitiesFieldProps
) {
  const { t } = useTranslation()
  const { fields, append, remove } = useFieldArray({
    control: props.control,
    name: 'video_model_capabilities',
  })

  return (
    <FormItem className='gap-3 px-4 py-3'>
      <div>
        <FormLabel>{t('Seedance model capabilities')}</FormLabel>
        <FormDescription>
          {t('Channel overrides for supported video resolutions')}
        </FormDescription>
      </div>

      {fields.length === 0 ? (
        <div className='text-muted-foreground flex min-h-16 items-center justify-center rounded-md border border-dashed px-3 text-center text-sm'>
          {t('Built-in Seedance defaults are active')}
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
                      <FormControl>
                        <Input
                          {...field}
                          placeholder='tvideos'
                          autoComplete='off'
                        />
                      </FormControl>
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
                      <ToggleGroup
                        multiple
                        value={field.value || []}
                        onValueChange={(value) =>
                          field.onChange(value as VideoResolution[])
                        }
                        variant='outline'
                        size='sm'
                        spacing={1}
                        className='grid w-full grid-cols-2 gap-1 sm:grid-cols-4'
                        aria-label={t('Supported resolutions')}
                      >
                        {VIDEO_RESOLUTIONS.map((resolution) => (
                          <ToggleGroupItem
                            key={resolution}
                            value={resolution}
                            className='w-full'
                          >
                            {resolution}
                          </ToggleGroupItem>
                        ))}
                      </ToggleGroup>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          ))}
        </div>
      )}

      <Button
        type='button'
        variant='outline'
        size='sm'
        className='w-full'
        onClick={() => append({ model: '', resolutions: [] })}
      >
        <Plus className='size-4' aria-hidden='true' />
        {t('Add model capability')}
      </Button>
    </FormItem>
  )
}
