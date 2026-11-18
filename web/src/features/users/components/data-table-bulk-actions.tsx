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
import { PowerOffIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { Table } from '@tanstack/react-table'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table/toolbar/bulk-actions'
import { Button } from '@/components/ui/button'

import { manageUser } from '../api'
import { isUserDeleted, USER_ROLE, USER_STATUS } from '../constants'
import type { User } from '../types'
import type { useUsers } from './users-provider'

interface DataTableBulkActionsProps {
  table: Table<User>
  requestVerification: ReturnType<typeof useUsers>['requestVerification']
  verificationActive?: boolean
}

export function DataTableBulkActions(props: DataTableBulkActionsProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [usersToDisable, setUsersToDisable] = useState<User[] | null>(null)
  const eligibleUsers = props.table
    .getFilteredSelectedRowModel()
    .rows.map((row) => row.original)
    .filter(
      (user) =>
        user.status === USER_STATUS.ENABLED &&
        user.role !== USER_ROLE.ROOT &&
        !isUserDeleted(user)
    )

  const disableMutation = useMutation({
    mutationFn: async (users: User[]) => {
      const succeededIds: number[] = []
      const failedIds: number[] = []

      for (const user of users) {
        let proof
        try {
          proof = await props.requestVerification({
            scope: 'admin.user.manage',
            context: { user_id: user.id, action: 'disable' },
            title: t('Disable selected users'),
            description: t(
              'Confirm your identity before changing the account {{username}}.',
              {
                username: user.username,
              }
            ),
          })
        } catch {
          break
        }
        if (!proof) break
        try {
          const result = await manageUser(user.id, 'disable', proof.proof_token)
          if (result.success) succeededIds.push(user.id)
          else failedIds.push(user.id)
        } catch {
          failedIds.push(user.id)
        }
      }

      return { succeededIds, failedIds }
    },
    onSuccess: async (result) => {
      props.table.setRowSelection((selection) => {
        const next = { ...selection }
        for (const id of result.succeededIds) {
          delete next[String(id)]
        }
        return next
      })

      if (result.succeededIds.length > 0) {
        toast.success(
          t('Disabled {{count}} user(s)', {
            count: result.succeededIds.length,
          })
        )
      }
      if (result.failedIds.length > 0) {
        toast.error(
          t('Failed to disable {{count}} user(s). Please try again.', {
            count: result.failedIds.length,
          })
        )
      }

      await queryClient.invalidateQueries({ queryKey: ['users'] })
      setUsersToDisable(null)
    },
  })

  return (
    <>
      <BulkActionsToolbar table={props.table} entityName='user'>
        <Button
          variant='destructive'
          size='sm'
          aria-label={t('Disable selected users')}
          disabled={eligibleUsers.length === 0 || disableMutation.isPending}
          onClick={() => setUsersToDisable(eligibleUsers)}
        >
          <HugeiconsIcon
            icon={PowerOffIcon}
            data-icon='inline-start'
            aria-hidden='true'
          />
          {t('Disable')}
        </Button>
      </BulkActionsToolbar>

      <ConfirmDialog
        open={usersToDisable !== null && !props.verificationActive}
        onOpenChange={(open) => {
          if (!open && !disableMutation.isPending) setUsersToDisable(null)
        }}
        title={t('Disable selected users')}
        desc={t(
          'Disable {{count}} selected user(s)? They will no longer be able to sign in or use the API. Already disabled, deleted, and root users are skipped.',
          { count: usersToDisable?.length ?? 0 }
        )}
        confirmText={
          disableMutation.isPending ? t('Disabling...') : t('Disable')
        }
        destructive
        isLoading={disableMutation.isPending}
        disabled={!usersToDisable?.length}
        handleConfirm={() => {
          if (usersToDisable?.length && !disableMutation.isPending) {
            disableMutation.mutate(usersToDisable)
          }
        }}
      />
    </>
  )
}
