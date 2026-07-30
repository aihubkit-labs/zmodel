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
import { Check, Copy, Download, Play } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { cn } from '@/lib/utils'

interface VideoPreviewDialogProps {
  taskId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function VideoPreviewDialog(props: VideoPreviewDialogProps) {
  const { t } = useTranslation()
  const [hasError, setHasError] = useState(false)
  const { copiedText, copyToClipboard } = useCopyToClipboard()
  const videoPath = `/v1/videos/${encodeURIComponent(props.taskId)}/content`
  const videoUrl =
    typeof window === 'undefined'
      ? videoPath
      : new URL(videoPath, window.location.origin).href
  const downloadUrl = `${videoUrl}?download_name=${encodeURIComponent(props.taskId)}`
  const isCopied = copiedText === videoUrl

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (open) setHasError(false)
        props.onOpenChange(open)
      }}
      title={
        <>
          <IconBadge tone='primary' size='sm'>
            <Play />
          </IconBadge>
          {t('Preview')}
        </>
      }
      titleClassName='flex items-center gap-2'
      contentClassName='w-[min(90vw,64rem)] sm:max-w-5xl'
      contentHeight='auto'
    >
      <div className='mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
        <p className='text-muted-foreground min-w-0 text-sm break-all'>
          {t('Task ID:')} {props.taskId}
        </p>
        <div className='flex flex-col gap-2 sm:flex-row'>
          <Button
            type='button'
            variant='outline'
            size='lg'
            className={cn(
              'h-10 min-w-40 px-5 max-sm:w-full',
              isCopied
                ? 'border-success/30 bg-success/10 text-success hover:bg-success/20 hover:text-success'
                : 'border-primary/30 bg-primary/10 text-primary hover:bg-primary/20 hover:text-primary'
            )}
            onClick={() => void copyToClipboard(videoUrl)}
          >
            {isCopied ? (
              <Check data-icon='inline-start' />
            ) : (
              <Copy data-icon='inline-start' />
            )}
            {t('Copy Link')}
          </Button>
          <Button
            size='lg'
            className='h-10 min-w-40 px-5 max-sm:w-full'
            render={<a href={downloadUrl} download={`${props.taskId}.mp4`} />}
          >
            <Download data-icon='inline-start' />
            {t('Download video')}
          </Button>
        </div>
      </div>
      <div className='bg-muted/30 flex min-h-64 items-center justify-center overflow-hidden rounded-md border'>
        {hasError ? (
          <p className='text-muted-foreground px-6 py-16 text-center text-sm'>
            {t('Failed to load')}
          </p>
        ) : (
          <video
            key={videoUrl}
            src={videoUrl}
            controls
            autoPlay
            playsInline
            preload='metadata'
            className='max-h-[calc(100dvh-14rem)] w-full bg-black object-contain'
            onError={() => setHasError(true)}
          />
        )}
      </div>
    </Dialog>
  )
}
