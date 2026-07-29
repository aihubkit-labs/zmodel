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
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { getObjectStorageSettings, updateObjectStorageSettings } from './api'
import {
  createObjectStorageSchema,
  type ObjectStorageFormValues,
} from './schema'

const initialValues: ObjectStorageFormValues = {
  endpoint: '',
  region: '',
  bucket: '',
  access_key: '',
  secret_access_key: '',
  retention_seconds: 86_400,
  presign_seconds: 600,
  archive_timeout_seconds: 600,
  archive_max_attempts: 8,
  archive_retry_window_seconds: 21_600,
  cleanup_interval_seconds: 900,
}

const numberFields = [
  ['retention_seconds', 'Object retention (seconds)'],
  ['presign_seconds', 'Presigned URL lifetime (seconds)'],
  ['archive_timeout_seconds', 'Archive attempt timeout (seconds)'],
  ['archive_max_attempts', 'Maximum archive attempts'],
  ['archive_retry_window_seconds', 'Maximum retry window (seconds)'],
  ['cleanup_interval_seconds', 'Cleanup interval (seconds)'],
] as const

const connectionFields = [
  ['region', 'Region'],
  ['bucket', 'Bucket'],
  ['access_key', 'Access Key'],
] as const

export function ObjectStorageSettingsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const settingsQuery = useQuery({
    queryKey: ['object-storage-settings'],
    queryFn: getObjectStorageSettings,
  })
  const form = useForm<ObjectStorageFormValues>({
    resolver: zodResolver(createObjectStorageSchema(t)),
    defaultValues: initialValues,
  })

  useEffect(() => {
    if (!settingsQuery.data) return
    form.reset({
      ...settingsQuery.data,
      secret_access_key: '',
    })
  }, [form, settingsQuery.data])

  const updateMutation = useMutation({
    mutationFn: updateObjectStorageSettings,
    onSuccess: async () => {
      toast.success(t('Object storage settings saved'))
      await queryClient.invalidateQueries({
        queryKey: ['object-storage-settings'],
      })
    },
    onError: () => toast.error(t('Failed to save object storage settings')),
  })

  const onSubmit = (values: ObjectStorageFormValues) => {
    updateMutation.mutate(values)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Object Storage')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          onClick={form.handleSubmit(onSubmit)}
          disabled={updateMutation.isPending || settingsQuery.isLoading}
        >
          {updateMutation.isPending ? t('Saving...') : t('Save settings')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-4xl flex-col gap-4'>
          <Alert>
            <AlertTitle>{t('Secret storage notice')}</AlertTitle>
            <AlertDescription>
              {t(
                'The S3 Secret Access Key is stored as plaintext in the database. Anyone with database or backup access can read it.'
              )}
            </AlertDescription>
          </Alert>

          <div className='rounded-xl border p-4 sm:p-6'>
            <div className='mb-5 flex items-center justify-between gap-3'>
              <div>
                <h2 className='font-semibold'>{t('S3 connection')}</h2>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Objects remain private and are delivered with temporary signed URLs.'
                  )}
                </p>
              </div>
              <Badge
                variant={
                  settingsQuery.data?.secret_configured
                    ? 'secondary'
                    : 'destructive'
                }
              >
                {settingsQuery.data?.secret_configured
                  ? t('Secret configured')
                  : t('Secret not configured')}
              </Badge>
            </div>

            <Form {...form}>
              <form
                onSubmit={form.handleSubmit(onSubmit)}
                className='grid gap-5 md:grid-cols-2'
                autoComplete='off'
              >
                <FormField
                  control={form.control}
                  name='endpoint'
                  render={({ field }) => (
                    <FormItem className='md:col-span-2'>
                      <FormLabel>{t('Endpoint URL')}</FormLabel>
                      <FormControl>
                        <Input
                          type='url'
                          placeholder='https://s3.example.com'
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Leave blank to use the standard AWS endpoint.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                {connectionFields.map(([name, label]) => (
                  <FormField
                    key={name}
                    control={form.control}
                    name={name}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t(label)}</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ))}
                <FormField
                  control={form.control}
                  name='secret_access_key'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Secret Access Key')}</FormLabel>
                      <FormControl>
                        <Input
                          type='password'
                          autoComplete='new-password'
                          placeholder={t(
                            'Leave blank to keep the existing secret'
                          )}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                {numberFields.map(([name, label]) => (
                  <FormField
                    key={name}
                    control={form.control}
                    name={name}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t(label)}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
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
              </form>
            </Form>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
