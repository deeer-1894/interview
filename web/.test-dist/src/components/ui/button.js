import { jsx as _jsx } from "react/jsx-runtime";
import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva } from "class-variance-authority";
import { cn } from "../../lib/utils.js";
const buttonVariants = cva("inline-flex items-center justify-center whitespace-nowrap rounded-full text-sm font-medium transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-45", {
    variants: {
        variant: {
            default: "bg-primary text-primary-foreground shadow-paper hover:-translate-y-0.5 hover:shadow-[0_18px_42px_rgba(30,95,255,0.18)]",
            secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/85",
            outline: "border border-border bg-background/88 backdrop-blur-sm hover:bg-accent hover:text-accent-foreground",
            ghost: "text-foreground hover:bg-accent/70 hover:text-accent-foreground",
        },
        size: {
            default: "h-11 px-5",
            sm: "h-9 px-4 text-xs",
            lg: "h-14 px-7 text-base",
            icon: "h-11 w-11",
        },
    },
    defaultVariants: {
        variant: "default",
        size: "default",
    },
});
const Button = React.forwardRef(({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return _jsx(Comp, { className: cn(buttonVariants({ variant, size, className })), ref: ref, ...props });
});
Button.displayName = "Button";
export { Button, buttonVariants };
