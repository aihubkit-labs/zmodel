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
export type AsyncImageTask = {
  task_id: string
  user_id: number
  username?: string
  model: string
  using_group: string
  channel_id?: number
  channel_name?: string
  platform?: string
  status: string
  output_availability: string
  billing_status: string
  object_available_count: number
  object_total_count: number
  archive_attempts: number
  staging_integrity: string
  error?: string
  created_at: number
  generation_completed_at: number
  completed_at: number
  output_expires_at: number
  manually_recovered_at: number
  reserved_quota?: number
  actual_quota?: number
}

export type AsyncImageTaskObject = {
  index: number
  status: string
  staging_status: string
  mime_type: string
  extension: string
  size_bytes: number
  staging_size_bytes: number
  uploaded_at: number
  expires_at: number
  staged_at: number
  deleted_at: number
  delete_attempts: number
  preview_url?: string
  download_url?: string
  url_unavailable?: boolean
  provider?: string
  endpoint?: string
  region?: string
  bucket?: string
  object_key?: string
  etag?: string
  last_error?: string
}

export type AsyncImageTaskDetail = {
  task_id: string
  user_id: number
  username?: string
  token_id?: number
  model: string
  channel_name?: string
  platform?: string
  status: string
  output_availability: string
  billing_status: string
  billing_source: string
  subscription_id?: number
  reserved_quota: number
  actual_quota: number
  using_group: string
  last_channel_id?: number
  request?: unknown
  retention_seconds: number
  archive_timeout_seconds: number
  archive_max_attempts: number
  archive_attempts: number
  archive_retry_deadline_at: number
  next_attempt_at: number
  source_kind: string
  error_code?: string
  error?: string
  created_at: number
  started_at: number
  generation_completed_at: number
  billing_finalized_at: number
  completed_at: number
  updated_at: number
  output_expires_at: number
  manually_recovered_at: number
  objects: AsyncImageTaskObject[]
}

export type AsyncImageTaskList = {
  items: AsyncImageTask[]
  total: number
  page: number
  page_size: number
}

export type RetryResult = {
  accepted_count: number
  skipped_count: number
  integrity_error_count: number
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}
