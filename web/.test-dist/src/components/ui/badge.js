import { jsx as _jsx } from "react/jsx-runtime";
import { cva } from "class-variance-authority";
import { cn } from "../../lib/utils.js";
const badgeVariants = cva("inline-flex items-center rounded-full border px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.2em] transition-colors", {
    variants: {
        variant: {
            default: "border-transparent bg-primary/12 text-primary",
            outline: "border-border/80 bg-background/70 text-foreground",
        },
    },
    defaultVariants: {
        variant: "default",
    },
});
export function Badge({ className, variant, ...props }) {
    return _jsx("div", { className: cn(badgeVariants({ variant }), className), ...props });
}
