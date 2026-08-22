'use client';

import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';

import { cn } from '@/lib/utils';

const Dialog = DialogPrimitive.Root;

const DialogTrigger = DialogPrimitive.Trigger;

const DialogPortal = DialogPrimitive.Portal;

const DialogClose = DialogPrimitive.Close;

const DialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    className={cn(
      'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/80',
      className
    )}
    {...props}
  />
));
DialogOverlay.displayName = DialogPrimitive.Overlay.displayName;

interface DialogContentProps extends React.ComponentPropsWithoutRef<
  typeof DialogPrimitive.Content
> {
  /**
   * Let the browser focus the first field on open, instead of the dialog itself.
   *
   * Opt in only where the dialog *is* one short input and the user's next act is
   * certainly to type into it -- an MFA code, a password confirmation. Those
   * forms already run `mode: 'onChange'` for the same reason.
   */
  autoFocusFirstField?: boolean;
}

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  DialogContentProps
>(({ className, children, autoFocusFirstField, onOpenAutoFocus, ...props }, ref) => {
  const t = useTranslations('common');
  return (
    <DialogPortal>
      <DialogOverlay />
      <DialogPrimitive.Content
        ref={ref}
        className={cn(
          // The original geometry, unchanged for every dialog that has not been
          // converted: a centred box that scrolls as a whole.
          'group bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid max-h-[85vh] w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 overflow-y-auto border p-6 shadow-lg duration-200 sm:w-full sm:max-w-lg sm:rounded-lg ' +
            // With a DialogBody it becomes three regions: header, scrolling body,
            // footer. The padding moves into them, so it is dropped here.
            'has-[[data-dialog-body]]:flex has-[[data-dialog-body]]:flex-col has-[[data-dialog-body]]:gap-0 has-[[data-dialog-body]]:overflow-hidden has-[[data-dialog-body]]:p-0 ' +
            // And below sm it becomes the screen: no centred box, no height cap,
            // no margins, so the keyboard has somewhere to go and there is only
            // one scroll container in play. 100dvh rather than 100vh, which
            // ignores the browser's collapsing chrome and leaves the bottom of the
            // layout under the URL bar.
            'max-sm:has-[[data-dialog-body]]:inset-0 max-sm:has-[[data-dialog-body]]:top-0 max-sm:has-[[data-dialog-body]]:left-0 max-sm:has-[[data-dialog-body]]:h-[100dvh] max-sm:has-[[data-dialog-body]]:max-h-none max-sm:has-[[data-dialog-body]]:w-full max-sm:has-[[data-dialog-body]]:max-w-none max-sm:has-[[data-dialog-body]]:translate-x-0 max-sm:has-[[data-dialog-body]]:translate-y-0 max-sm:has-[[data-dialog-body]]:rounded-none max-sm:has-[[data-dialog-body]]:border-0',
          className
        )}
        // Focus the dialog, not the first field.
        //
        // Radix focuses the first focusable descendant on open, which puts the
        // cursor in a field the user did not choose. Forms here validate
        // `onTouched` -- a field is judged once the user has finished with it --
        // so their first click anywhere else *blurs* that field and it is judged
        // for a value they were never given the chance to enter. Opening Create
        // Child and adding a property first answered with "First name is
        // required" and a red "Please correct 1 entry" above a form barely
        // started, which is exactly the "told you are wrong before you had a
        // chance to be right" behaviour validation-timing.ts exists to prevent.
        //
        // Focusing the container is also the better announcement: a screen
        // reader reads the dialog's title and role rather than starting halfway
        // in at an unlabelled-in-context input.
        onOpenAutoFocus={(event) => {
          onOpenAutoFocus?.(event);
          if (autoFocusFirstField || event.defaultPrevented) return;
          event.preventDefault();
          // Radix gives Content tabIndex={-1}, so it takes focus without
          // becoming a tab stop.
          (event.currentTarget as HTMLElement | null)?.focus();
        }}
        {...props}
      >
        {children}
        {/* 44px, not a bare 16px icon. Below `sm` these dialogs are the whole
          screen, which makes this the primary way out of one — it has to meet
          the same touch minimum as everything else, and its label has to come
          from the catalogue like every other string the user hears. The icon
          keeps its 16px drawing; only the hit area grows around it. */}
        <DialogPrimitive.Close
          aria-label={t('close')}
          className="ring-offset-background focus-visible:ring-ring data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute top-2 right-2 inline-flex h-11 w-11 items-center justify-center rounded-md opacity-70 transition-opacity hover:opacity-100 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:pointer-events-none"
        >
          <X className="h-4 w-4" />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPortal>
  );
});
DialogContent.displayName = DialogPrimitive.Content.displayName;

/**
 * The scrolling region of a dialog.
 *
 * Wrap the fields in this and leave DialogHeader and DialogFooter outside it.
 * Everything then holds regardless of how much content there is: the header and
 * the actions stay put, and only the middle moves.
 *
 * It is a component rather than something DialogContent infers from its
 * children, because inference does not survive the shape these dialogs actually
 * have — the footer usually sits inside a <form>, where partitioning by child
 * type silently fails to find it and the layout quietly reverts.
 *
 * overscroll-contain stops a scroll that reaches the end of this region from
 * chaining to the page behind the dialog.
 */
const DialogBody = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    data-dialog-body=""
    className={cn('min-h-0 flex-1 overflow-y-auto overscroll-contain px-6 py-2', className)}
    {...props}
  />
);
DialogBody.displayName = 'DialogBody';

const DialogHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      'flex flex-col space-y-1.5 text-center sm:text-left',
      // Padding lives on the container in the old layout and on the regions in
      // the new one, so it applies here only when a DialogBody is present.
      'group-has-[[data-dialog-body]]:shrink-0 group-has-[[data-dialog-body]]:px-6 group-has-[[data-dialog-body]]:pt-6 group-has-[[data-dialog-body]]:pb-4',
      className
    )}
    {...props}
  />
);
DialogHeader.displayName = 'DialogHeader';

const DialogFooter = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      // A sibling of the scrolling body, never inside it: it cannot be pushed
      // out of reach by content growing above it, and it cannot cover anything.
      // The safe-area padding matters on a phone, where this sits at the very
      // bottom of the screen over the home indicator.
      'flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2',
      // In a three-region dialog this is a sibling of the scrolling body, never
      // inside it: it cannot be pushed out of reach by content growing above it,
      // and it cannot cover anything. The safe-area padding matters on a phone,
      // where it sits at the bottom of the screen over the home indicator. In
      // the old layout none of this applies.
      'group-has-[[data-dialog-body]]:bg-background group-has-[[data-dialog-body]]:shrink-0 group-has-[[data-dialog-body]]:gap-2 group-has-[[data-dialog-body]]:border-t group-has-[[data-dialog-body]]:px-6 group-has-[[data-dialog-body]]:py-4 group-has-[[data-dialog-body]]:pb-[max(1rem,env(safe-area-inset-bottom))] sm:group-has-[[data-dialog-body]]:pb-4',
      className
    )}
    {...props}
  />
);
DialogFooter.displayName = 'DialogFooter';

const DialogTitle = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title
    ref={ref}
    className={cn('pr-8 text-lg leading-none font-semibold tracking-tight', className)}
    {...props}
  />
));
DialogTitle.displayName = DialogPrimitive.Title.displayName;

const DialogDescription = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description
    ref={ref}
    className={cn('text-muted-foreground pr-8 text-sm', className)}
    {...props}
  />
));
DialogDescription.displayName = DialogPrimitive.Description.displayName;

export {
  Dialog,
  DialogBody,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
};
