import { jsx as _jsx } from "react/jsx-runtime";
import { cn } from "../../lib/utils.js";
export function Separator({ className }) {
    return _jsx("div", { className: cn("h-px w-full bg-border/80", className) });
}
