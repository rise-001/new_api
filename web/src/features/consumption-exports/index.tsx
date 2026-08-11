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
import { Download04Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Main } from '@/components/layout'
import { Button } from '@/components/ui/button'

import { createConsumptionExport } from './api'
import { CreateExportDialog } from './components/create-export-dialog'
import type {
  ConsumptionExportDownload,
  CreateConsumptionExportInput,
} from './types'

function saveConsumptionExport(download: ConsumptionExportDownload) {
  const objectURL = URL.createObjectURL(download.blob)
  const link = document.createElement('a')
  link.href = objectURL
  link.download = download.fileName
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectURL), 0)
}

export function ConsumptionExports() {
  const { t } = useTranslation()
  const [createOpen, setCreateOpen] = useState(false)
  const createMutation = useMutation({ mutationFn: createConsumptionExport })

  const handleCreate = async (
    input: CreateConsumptionExportInput
  ): Promise<boolean> => {
    try {
      const download = await createMutation.mutateAsync(input)
      saveConsumptionExport(download)
      return true
    } catch (error) {
      if (
        error instanceof Error &&
        error.message === 'An export is already being generated'
      ) {
        toast.error(t('An export is already being generated'))
      } else if (
        error instanceof Error &&
        error.message === 'Download failed'
      ) {
        toast.error(t('Download failed'))
      } else {
        toast.error(
          error instanceof Error ? error.message : t('Download failed')
        )
      }
      return false
    }
  }

  return (
    <Main>
      <div className='min-h-0 flex-1 overflow-auto px-3 py-3 sm:px-4 sm:py-6'>
        <section className='mx-auto w-full max-w-[1600px]'>
          <header className='flex flex-col gap-4 border-b pb-4 sm:flex-row sm:items-center sm:justify-between'>
            <h1 className='text-base font-semibold'>
              {t('Consumption exports')}
            </h1>
            <Button type='button' onClick={() => setCreateOpen(true)}>
              <HugeiconsIcon
                icon={Download04Icon}
                strokeWidth={2}
                data-icon='inline-start'
              />
              {t('Create export')}
            </Button>
          </header>
        </section>
      </div>

      <CreateExportDialog
        open={createOpen}
        pending={createMutation.isPending}
        onOpenChange={setCreateOpen}
        onCreate={handleCreate}
      />
    </Main>
  )
}
