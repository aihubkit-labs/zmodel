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
  model: string
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
