'use client';

import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
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

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content>
>(({ className, children, ...props }, ref) => (
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
      {...props}
    >
      {children}
      <DialogPrimitive.Close className="ring-offset-background focus:ring-ring data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute top-4 right-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-none disabled:pointer-events-none">
        <X className="h-4 w-4" />
        <span className="sr-only">Close</span>
      </DialogPrimitive.Close>
    </DialogPrimitive.Content>
  </DialogPortal>
));
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
    className={cn('text-lg leading-none font-semibold tracking-tight', className)}
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
    className={cn('text-muted-foreground text-sm', className)}
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
