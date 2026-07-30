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
export type ObjectStorageSettings = {
  endpoint: string
  region: string
  bucket: string
  access_key: string
  secret_configured: boolean
  staging_directory: string
  retention_seconds: number
  presign_seconds: number
  archive_timeout_seconds: number
  archive_max_attempts: number
  archive_retry_window_seconds: number
  cleanup_interval_seconds: number
}

export type UpdateObjectStorageSettings = Omit<
  ObjectStorageSettings,
  'secret_configured'
> & {
  secret_access_key: string
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}
