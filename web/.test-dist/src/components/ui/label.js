import { jsx as _jsx } from "react/jsx-runtime";
import * as React from "react";
import * as LabelPrimitive from "@radix-ui/react-label";
import { cn } from "../../lib/utils.js";
const Label = React.forwardRef(({ className, ...props }, ref) => (_jsx(LabelPrimitive.Root, { ref: ref, className: cn("text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground", className), ...props })));
Label.displayName = LabelPrimitive.Root.displayName;
export { Label };
