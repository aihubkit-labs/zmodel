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
import * as z from 'zod'

export const createObjectStorageSchema = (t: (key: string) => string) =>
  z
    .object({
      endpoint: z
        .string()
        .trim()
        .refine(
          (value) => value === '' || /^https?:\/\//.test(value),
          t('Provide a valid URL starting with http:// or https://')
        ),
      region: z.string().trim().min(1, t('Region is required')),
      bucket: z
        .string()
        .trim()
        .min(1, t('Bucket is required'))
        .refine(
          (value) => !/\s/.test(value),
          t('Bucket cannot contain spaces')
        ),
      access_key: z.string().trim().min(1, t('Access Key is required')),
      secret_access_key: z.string(),
      staging_directory: z
        .string()
        .trim()
        .min(1, t('Persistent staging directory is required'))
        .refine(
          (value) => value.startsWith('/'),
          t('Use an absolute server path starting with /')
        ),
      retention_seconds: z.number().int().min(60).max(31_536_000),
      presign_seconds: z.number().int().min(60).max(604_800),
      archive_timeout_seconds: z.number().int().min(1).max(1_200),
      archive_max_attempts: z.number().int().min(1).max(100),
      archive_retry_window_seconds: z.number().int().min(60).max(604_800),
      cleanup_interval_seconds: z.number().int().min(60).max(86_400),
    })
    .refine(
      (values) =>
        values.archive_retry_window_seconds >= values.archive_timeout_seconds,
      {
        path: ['archive_retry_window_seconds'],
        message: t('Retry window must not be shorter than archive timeout'),
      }
    )

export type ObjectStorageFormValues = z.infer<
  ReturnType<typeof createObjectStorageSchema>
>
