import { jsx as _jsx } from "react/jsx-runtime";
import * as React from "react";
import { cn } from "../../lib/utils.js";
const Textarea = React.forwardRef(({ className, ...props }, ref) => {
    return (_jsx("textarea", { className: cn("flex min-h-[140px] w-full rounded-[1.5rem] border border-input bg-background/75 px-4 py-3 text-sm text-foreground shadow-sm transition placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50", className), ref: ref, ...props }));
});
Textarea.displayName = "Textarea";
export { Textarea };
