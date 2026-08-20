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
/* oxlint-disable react/iframe-missing-sandbox -- Embedded admin-configured apps require scripts and origin-scoped storage. */
import { LinkSquare01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useCustomMenus } from '@/hooks/use-custom-menus'
import { CUSTOM_MENU_LOCATION, CUSTOM_MENU_OPEN_MODE } from '@/lib/custom-menus'

export const Route = createFileRoute('/_authenticated/custom-menu/$menuId')({
  component: CustomMenuEmbedPage,
})

function CustomMenuEmbedPage() {
  const { t } = useTranslation()
  const { menuId } = Route.useParams()
  const items = useCustomMenus()
  const item = items.find(
    (candidate) =>
      candidate.id === menuId &&
      (candidate.location === CUSTOM_MENU_LOCATION.CHAT ||
        candidate.location === CUSTOM_MENU_LOCATION.PERSONAL) &&
      candidate.openMode === CUSTOM_MENU_OPEN_MODE.EMBED
  )

  if (!item) {
    return (
      <div className='flex h-full flex-col items-center justify-center gap-4 p-6 text-center'>
        <HugeiconsIcon
          icon={LinkSquare01Icon}
          strokeWidth={1.5}
          className='text-muted-foreground size-12'
        />
        <div className='space-y-1'>
          <h2 className='text-lg font-semibold'>
            {t('Custom menu not found')}
          </h2>
          <p className='text-muted-foreground text-sm'>
            {t(
              'The requested custom menu does not exist or is no longer embedded.'
            )}
          </p>
        </div>
        <Button variant='outline' render={<Link to='/dashboard' />}>
          {t('Return to dashboard')}
        </Button>
      </div>
    )
  }

  return (
    <iframe
      src={item.url}
      title={item.name}
      referrerPolicy='strict-origin-when-cross-origin'
      className='h-full w-full border-0'
      allow='clipboard-read; clipboard-write; fullscreen'
      sandbox='allow-downloads allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts'
    />
  )
}
