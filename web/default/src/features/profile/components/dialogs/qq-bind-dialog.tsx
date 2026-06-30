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
import { useState, useEffect } from 'react'
import { Loader2, MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { getQqBindCode, confirmQqBind } from '../../api'

interface QqBindDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function QqBindDialog({
  open,
  onOpenChange,
  onSuccess,
}: QqBindDialogProps) {
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const [expireSeconds, setExpireSeconds] = useState(0)
  const [loading, setLoading] = useState(false)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    if (!open) {
      setCode('')
      setExpireSeconds(0)
      return
    }

    setLoading(true)
    getQqBindCode()
      .then((res) => {
        if (res.success && res.data) {
          setCode(res.data.code)
          setExpireSeconds(res.data.expire_seconds)
        } else {
          toast.error(res.message || t('Failed to generate code'))
        }
      })
      .catch(() => {
        toast.error(t('Failed to generate code'))
      })
      .finally(() => {
        setLoading(false)
      })
  }, [open, t])

  const handleConfirm = async () => {
    setConfirming(true)
    try {
      const res = await confirmQqBind()
      if (res.success) {
        toast.success(t('QQ account bound successfully'))
        onSuccess()
        onOpenChange(false)
      } else {
        toast.error(res.message || t('Failed to bind QQ'))
      }
    } catch {
      toast.error(t('Failed to bind QQ'))
    } finally {
      setConfirming(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Bind QQ Account')}
      description={t('Send this code to the QQ bot to complete binding')}
      contentClassName='sm:max-w-md'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      <div className='space-y-4 py-4'>
        <Alert>
          <MessageCircle className='h-4 w-4' />
          <AlertDescription>
            {t(
              'Bind your QQ account to enable API access'
            )}
          </AlertDescription>
        </Alert>

        <div className='flex flex-col items-center justify-center gap-4 rounded-lg border p-6'>
          <div className='flex h-12 w-12 items-center justify-center rounded-xl bg-green-100 dark:bg-green-900'>
            <MessageCircle className='h-6 w-6 text-green-600 dark:text-green-400' />
          </div>

          <div className='text-center'>
            {loading ? (
              <p className='text-muted-foreground text-sm'>
                {t('Generating code...')}
              </p>
            ) : (
              <>
                <p className='font-mono text-2xl font-bold tracking-widest'>
                  {code}
                </p>
                <p className='text-muted-foreground mt-1 text-xs'>
                  {expireSeconds > 0 &&
                    t('Expires in {{seconds}} seconds', {
                      seconds: String(expireSeconds),
                    })}
                </p>
              </>
            )}
          </div>

          <p className='text-muted-foreground text-center text-sm'>
            {t('Send this code to the QQ bot')}
          </p>
        </div>

        <Button
          className='w-full'
          onClick={handleConfirm}
          disabled={loading || confirming || !code}
        >
          {confirming && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
          {confirming ? t('Binding...') : t('Confirm binding')}
        </Button>
      </div>
    </Dialog>
  )
}
