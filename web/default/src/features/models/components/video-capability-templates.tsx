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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import {
  Copy,
  ExternalLink,
  Eye,
  Loader2,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { sideDrawerContentClassName } from '@/components/drawer-layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { formatTimestampToDate } from '@/lib/format'

import {
  deleteVideoModelCapabilityTemplate,
  getVideoModelCapabilityTemplates,
  saveVideoModelCapabilityTemplate,
} from '../../channels/api'
import { VideoModelCapabilitiesField } from '../../channels/components/video-model-capabilities-field'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  type ChannelFormValues,
} from '../../channels/lib'
import {
  defaultVideoModelCapability,
  videoModelCapabilityFromTemplate,
} from '../../channels/lib/video-model-capability'
import type {
  VideoModelCapability,
  VideoModelCapabilityTemplate,
  VideoProtocol,
} from '../../channels/types'

const route = getRouteApi('/_authenticated/models/$section')
const TEMPLATE_PROTOCOLS: VideoProtocol[] = ['megabyai', 'globalaiopc']

const templateIdentitySchema = z.object({
  video_protocol: z.enum(['megabyai', 'globalaiopc']),
  model_id: z.string().trim().min(1).max(128),
  name: z.string().trim().min(1).max(256),
  source_url: z
    .string()
    .trim()
    .refine(
      (value) =>
        !value ||
        (() => {
          try {
            const url = new URL(value)
            return url.protocol === 'http:' || url.protocol === 'https:'
          } catch {
            return false
          }
        })(),
      'Source URL must be a valid HTTP or HTTPS URL'
    ),
})

type TemplateIdentityValues = z.infer<typeof templateIdentitySchema>
type TemplateEditorState = {
  template: VideoModelCapabilityTemplate | null
  copyFrom: VideoModelCapabilityTemplate | null
  readOnly: boolean
}

function protocolLabel(protocol: VideoProtocol): string {
  return protocol || '-'
}

function capabilitySummary(capability: VideoModelCapability): string {
  const parts = [
    capability.resolutions.join(', '),
    capability.ratios?.length ? capability.ratios.join(', ') : null,
    capability.supports_duration
      ? `${capability.min_duration_seconds}-${capability.max_duration_seconds}s`
      : null,
  ]
  return parts.filter(Boolean).join(' · ')
}

function VideoCapabilityTemplateDrawer(props: {
  state: TemplateEditorState
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const source = props.state.template || props.state.copyFrom
  const editing = Boolean(props.state.template)
  const readOnly = props.state.readOnly
  let modelID = ''
  let name = ''
  if (editing) {
    modelID = source?.model_id || ''
    name = source?.name || ''
  } else if (source) {
    modelID = `${source.model_id}-copy`
    name = `${source.name} copy`
  }
  const identityForm = useForm<TemplateIdentityValues>({
    resolver: zodResolver(templateIdentitySchema),
    defaultValues: {
      video_protocol:
        source?.video_protocol === 'megabyai' ? 'megabyai' : 'globalaiopc',
      model_id: modelID,
      name,
      source_url: source?.source_url || '',
    },
  })
  const capabilityForm = useForm<ChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: {
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'video-capability-template',
      key: 'template',
      models: modelID || 'template-model',
      group: ['default'],
      video_protocol:
        source?.video_protocol === 'megabyai' ? 'megabyai' : 'globalaiopc',
      video_model_capabilities: [
        source
          ? videoModelCapabilityFromTemplate(source, modelID)
          : defaultVideoModelCapability('template-model'),
      ],
    },
  })

  const mutation = useMutation({
    mutationFn: saveVideoModelCapabilityTemplate,
    onSuccess: async (response) => {
      if (!response.success) {
        throw new Error(response.message || 'Save failed')
      }
      await queryClient.invalidateQueries({
        queryKey: ['video-capability-templates'],
      })
      toast.success(t('Video capability template saved'))
      props.onOpenChange(false)
    },
    onError: (error) => toast.error(error.message),
  })

  let title = t('Create video capability template')
  if (readOnly) {
    title = t('View video capability template')
  } else if (editing) {
    title = t('Edit video capability template')
  } else if (props.state.copyFrom) {
    title = t('Copy video capability template')
  }

  const handleSubmit = async () => {
    const identityValid = await identityForm.trigger()
    const capabilityValid = await capabilityForm.trigger()
    if (!identityValid || !capabilityValid) return

    const identity = identityForm.getValues()
    const capabilityValue = capabilityForm.getValues(
      'video_model_capabilities.0'
    )
    if (!capabilityValue) return
    const { model: _model, ...capability } = capabilityValue
    mutation.mutate({
      id: props.state.template?.id || 0,
      video_protocol: identity.video_protocol,
      model_id: identity.model_id,
      name: identity.name,
      capability: capability as VideoModelCapability,
      source: 'manual',
      source_url: identity.source_url || undefined,
      built_in: false,
    })
  }

  const models = [
    identityForm.watch('model_id') || source?.model_id || 'template-model',
  ]

  return (
    <Sheet open onOpenChange={props.onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName(
          'sm:max-w-2xl lg:max-w-3xl xl:max-w-4xl'
        )}
      >
        <SheetHeader className='border-b'>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>
            {t(
              'Templates provide reusable defaults for channel video model capabilities.'
            )}
          </SheetDescription>
        </SheetHeader>
        <div className='flex-1 space-y-5 overflow-y-auto px-4 pb-6'>
          <div className='grid gap-4 rounded-lg border p-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label>{t('Video Protocol')}</Label>
              <Select
                items={TEMPLATE_PROTOCOLS.map((protocol) => ({
                  value: protocol,
                  label: protocolLabel(protocol),
                }))}
                value={identityForm.watch('video_protocol')}
                disabled={readOnly}
                onValueChange={(value) => {
                  if (value === 'megabyai' || value === 'globalaiopc') {
                    identityForm.setValue('video_protocol', value, {
                      shouldValidate: true,
                    })
                    capabilityForm.setValue('video_protocol', value)
                  }
                }}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {TEMPLATE_PROTOCOLS.map((protocol) => (
                      <SelectItem key={protocol} value={protocol}>
                        {protocolLabel(protocol)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='template-model-id'>
                {t('Upstream model ID')}
              </Label>
              <Input
                id='template-model-id'
                disabled={readOnly}
                {...identityForm.register('model_id', {
                  onChange: (event) => {
                    capabilityForm.setValue(
                      'video_model_capabilities.0.model',
                      event.target.value,
                      { shouldValidate: true }
                    )
                  },
                })}
              />
              <p className='text-destructive text-xs'>
                {identityForm.formState.errors.model_id?.message}
              </p>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='template-name'>{t('Template name')}</Label>
              <Input
                id='template-name'
                disabled={readOnly}
                {...identityForm.register('name')}
              />
              <p className='text-destructive text-xs'>
                {identityForm.formState.errors.name?.message}
              </p>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='template-source-url'>
                {t('Source document')}
              </Label>
              <Input
                id='template-source-url'
                disabled={readOnly}
                placeholder='https://docs.example.com/video-model'
                {...identityForm.register('source_url')}
              />
              <p className='text-destructive text-xs'>
                {identityForm.formState.errors.source_url?.message}
              </p>
            </div>
          </div>

          <FormProvider {...capabilityForm}>
            <VideoModelCapabilitiesField
              control={capabilityForm.control}
              models={models}
              protocol={identityForm.watch('video_protocol')}
              templateToolsEnabled={false}
              readOnly={readOnly}
            />
          </FormProvider>
        </div>
        <SheetFooter className='border-t sm:flex-row sm:justify-end'>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          {!readOnly ? (
            <Button disabled={mutation.isPending} onClick={handleSubmit}>
              {mutation.isPending ? (
                <Loader2 className='size-4 animate-spin' aria-hidden='true' />
              ) : null}
              {t('Save')}
            </Button>
          ) : null}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

export function VideoCapabilityTemplates() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const queryClient = useQueryClient()
  const [editorState, setEditorState] = useState<TemplateEditorState | null>(
    null
  )
  const [deleteTemplate, setDeleteTemplate] =
    useState<VideoModelCapabilityTemplate | null>(null)

  const templatesQuery = useQuery({
    queryKey: ['video-capability-templates'],
    queryFn: () => getVideoModelCapabilityTemplates(''),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteVideoModelCapabilityTemplate,
    onSuccess: async (response) => {
      if (!response.success) {
        throw new Error(response.message || 'Delete failed')
      }
      await queryClient.invalidateQueries({
        queryKey: ['video-capability-templates'],
      })
      toast.success(t('Video capability template deleted'))
      setDeleteTemplate(null)
    },
    onError: (error) => toast.error(error.message),
  })

  const protocolFilter = search.protocol || 'all'
  const textFilter = search.templateFilter?.trim().toLowerCase() || ''
  const templates = useMemo(() => {
    const items = templatesQuery.data?.data || []
    return items.filter((template) => {
      if (
        protocolFilter !== 'all' &&
        template.video_protocol !== protocolFilter
      ) {
        return false
      }
      if (!textFilter) return true
      return `${template.name} ${template.model_id}`
        .toLowerCase()
        .includes(textFilter)
    })
  }, [protocolFilter, templatesQuery.data, textFilter])
  let editorKey = 'create-template'
  if (editorState?.template) {
    editorKey = `template-${editorState.template.id}-${editorState.readOnly ? 'view' : 'edit'}`
  } else if (editorState?.copyFrom) {
    editorKey = `copy-${editorState.copyFrom.id}`
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-4'>
      <div className='flex shrink-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex flex-1 flex-col gap-2 sm:flex-row'>
          <Input
            className='sm:max-w-sm'
            placeholder={t('Search template name or model ID')}
            value={search.templateFilter || ''}
            onChange={(event) =>
              void navigate({
                search: (previous) => ({
                  ...previous,
                  templateFilter: event.target.value,
                }),
                replace: true,
              })
            }
          />
          <Select
            items={[
              { value: 'all', label: t('All video protocols') },
              ...TEMPLATE_PROTOCOLS.map((protocol) => ({
                value: protocol,
                label: protocolLabel(protocol),
              })),
            ]}
            value={protocolFilter}
            onValueChange={(value) =>
              void navigate({
                search: (previous) => ({
                  ...previous,
                  protocol: value || 'all',
                }),
                replace: true,
              })
            }
          >
            <SelectTrigger className='w-full sm:w-52'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value='all'>{t('All video protocols')}</SelectItem>
                {TEMPLATE_PROTOCOLS.map((protocol) => (
                  <SelectItem key={protocol} value={protocol}>
                    {protocolLabel(protocol)}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <Button
          size='sm'
          onClick={() =>
            setEditorState({
              template: null,
              copyFrom: null,
              readOnly: false,
            })
          }
        >
          <Plus className='size-4' aria-hidden='true' />
          {t('Create template')}
        </Button>
      </div>

      <StaticDataTable
        className='min-h-0 flex-1 overflow-auto'
        data={templates}
        getRowKey={(template) => template.id}
        empty={templatesQuery.isLoading || templates.length === 0}
        emptyContent={
          templatesQuery.isLoading ? t('Loading...') : t('No templates found')
        }
        columns={[
          {
            id: 'name',
            header: t('Template name'),
            cell: (template) => (
              <div className='space-y-1'>
                <div className='font-medium'>{template.name}</div>
                <div className='text-muted-foreground font-mono text-xs'>
                  {template.model_id}
                </div>
              </div>
            ),
          },
          {
            id: 'protocol',
            header: t('Video Protocol'),
            cell: (template) => (
              <Badge variant='outline'>
                {protocolLabel(template.video_protocol)}
              </Badge>
            ),
          },
          {
            id: 'source',
            header: t('Template source'),
            cell: (template) => (
              <div className='flex items-center gap-2'>
                <Badge variant={template.built_in ? 'secondary' : 'outline'}>
                  {template.built_in
                    ? t('Built-in template')
                    : t('Custom template')}
                </Badge>
                {template.source_url ? (
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    aria-label={t('Source document')}
                    render={
                      <a
                        href={template.source_url}
                        target='_blank'
                        rel='noreferrer'
                      />
                    }
                  >
                    <ExternalLink aria-hidden='true' />
                  </Button>
                ) : null}
              </div>
            ),
          },
          {
            id: 'capability',
            header: t('Capability summary'),
            cell: (template) => capabilitySummary(template.capability),
          },
          {
            id: 'updated',
            header: t('Updated time'),
            cell: (template) => formatTimestampToDate(template.updated_time),
          },
          {
            id: 'actions',
            header: t('Actions'),
            className: 'w-36 text-right',
            cellClassName: 'text-right',
            cell: (template) => (
              <div className='flex justify-end gap-1'>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('View')}
                  title={t('View')}
                  onClick={() =>
                    setEditorState({
                      template,
                      copyFrom: null,
                      readOnly: true,
                    })
                  }
                >
                  <Eye aria-hidden='true' />
                </Button>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Copy')}
                  title={t('Copy')}
                  onClick={() =>
                    setEditorState({
                      template: null,
                      copyFrom: template,
                      readOnly: false,
                    })
                  }
                >
                  <Copy aria-hidden='true' />
                </Button>
                {!template.built_in ? (
                  <>
                    <Button
                      variant='ghost'
                      size='icon-sm'
                      aria-label={t('Edit')}
                      title={t('Edit')}
                      onClick={() =>
                        setEditorState({
                          template,
                          copyFrom: null,
                          readOnly: false,
                        })
                      }
                    >
                      <Pencil aria-hidden='true' />
                    </Button>
                    <Button
                      variant='ghost'
                      size='icon-sm'
                      className='text-destructive hover:text-destructive'
                      aria-label={t('Delete')}
                      title={t('Delete')}
                      onClick={() => setDeleteTemplate(template)}
                    >
                      <Trash2 aria-hidden='true' />
                    </Button>
                  </>
                ) : null}
              </div>
            ),
          },
        ]}
      />

      {editorState ? (
        <VideoCapabilityTemplateDrawer
          key={editorKey}
          state={editorState}
          onOpenChange={(open) => {
            if (!open) setEditorState(null)
          }}
        />
      ) : null}
      <ConfirmDialog
        open={Boolean(deleteTemplate)}
        onOpenChange={(open) => {
          if (!open) setDeleteTemplate(null)
        }}
        title={t('Delete video capability template')}
        desc={t(
          'Are you sure you want to delete template "{{name}}"? This action cannot be undone.',
          { name: deleteTemplate?.name || '' }
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (deleteTemplate) deleteMutation.mutate(deleteTemplate.id)
        }}
      />
    </div>
  )
}
