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
import { Add01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { nanoid } from 'nanoid'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  CUSTOM_MENU_LOCATION,
  CUSTOM_MENU_MAX_ITEMS,
  CUSTOM_MENU_OPEN_MODE,
  serializeCustomMenuItems,
  type CustomMenuItem,
} from '@/lib/custom-menus'

import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  CustomMenuDialog,
  type CustomMenuFormValues,
} from './custom-menu-dialog'

type CustomMenuSectionProps = {
  initialItems: CustomMenuItem[]
}

export function CustomMenuSection(props: CustomMenuSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [items, setItems] = useState(props.initialItems)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<CustomMenuItem | null>(null)
  const [deleteItem, setDeleteItem] = useState<CustomMenuItem | null>(null)

  const initialSerialized = serializeCustomMenuItems(props.initialItems)
  const serialized = serializeCustomMenuItems(items)
  const isDirty = serialized !== initialSerialized

  const handleAdd = () => {
    setEditingItem(null)
    setDialogOpen(true)
  }

  const handleEdit = (item: CustomMenuItem) => {
    setEditingItem(item)
    setDialogOpen(true)
  }

  const handleSubmit = (values: CustomMenuFormValues) => {
    if (editingItem) {
      setItems((current) =>
        current.map((item) =>
          item.id === editingItem.id ? { ...item, ...values } : item
        )
      )
    } else {
      setItems((current) => [
        ...current,
        {
          id: nanoid(10),
          ...values,
        },
      ])
    }
    setDialogOpen(false)
    setEditingItem(null)
  }

  const handleSave = async () => {
    if (!isDirty) return
    await updateOption.mutateAsync({
      key: 'CustomMenuItems',
      value: serialized,
    })
  }

  return (
    <SettingsSection title={t('Custom Menu Management')}>
      <SettingsPageFormActions
        onSave={handleSave}
        onReset={() => setItems(props.initialItems)}
        isSaving={updateOption.isPending}
        isSaveDisabled={!isDirty}
        isResetDisabled={!isDirty}
        saveLabel='Save custom menus'
      />

      <div className='flex items-center justify-end'>
        <Button
          size='sm'
          onClick={handleAdd}
          disabled={items.length >= CUSTOM_MENU_MAX_ITEMS}
        >
          <HugeiconsIcon
            icon={Add01Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          {t('Add custom menu')}
        </Button>
      </div>

      <StaticDataTable
        data={items}
        getRowKey={(item) => item.id}
        emptyContent={t('No custom menus configured.')}
        columns={[
          {
            id: 'name',
            header: t('Name'),
            cellClassName: 'font-medium',
            cell: (item) => item.name,
          },
          {
            id: 'url',
            header: t('URL'),
            cellClassName: 'max-w-md',
            cell: (item) => (
              <a
                href={item.url}
                target='_blank'
                rel='noopener noreferrer'
                title={item.url}
                className='text-muted-foreground hover:text-foreground block truncate font-mono text-xs'
              >
                {item.url}
              </a>
            ),
          },
          {
            id: 'location',
            header: t('Display area'),
            cell: (item) => (
              <Badge variant='outline'>
                {item.location === CUSTOM_MENU_LOCATION.CHAT
                  ? t('Chat area')
                  : t('Top navigation')}
              </Badge>
            ),
          },
          {
            id: 'open-mode',
            header: t('Open mode'),
            cell: (item) => (
              <Badge variant='secondary'>
                {item.openMode === CUSTOM_MENU_OPEN_MODE.EMBED
                  ? t('Embedded')
                  : t('New tab')}
              </Badge>
            ),
          },
          {
            id: 'actions',
            header: t('Actions'),
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (item) => (
              <StaticRowActions
                editLabel={t('Edit')}
                deleteLabel={t('Delete')}
                menuLabel={t('Open menu')}
                onEdit={() => handleEdit(item)}
                onDelete={() => setDeleteItem(item)}
              />
            ),
          },
        ]}
      />

      {dialogOpen && (
        <CustomMenuDialog
          item={editingItem}
          onOpenChange={setDialogOpen}
          onSubmit={handleSubmit}
        />
      )}

      <AlertDialog
        open={deleteItem !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteItem(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Delete custom menu?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('Remove {{name}} from the navigation menus?', {
                name: deleteItem?.name ?? '',
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={() => {
                if (deleteItem) {
                  setItems((current) =>
                    current.filter((item) => item.id !== deleteItem.id)
                  )
                }
                setDeleteItem(null)
              }}
            >
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
