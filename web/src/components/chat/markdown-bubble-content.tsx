import type { ComponentPropsWithoutRef } from "react";
import ReactMarkdown from "react-markdown";
import { PrismAsyncLight as SyntaxHighlighter } from "react-syntax-highlighter";
import bash from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import diff from "react-syntax-highlighter/dist/esm/languages/prism/diff";
import go from "react-syntax-highlighter/dist/esm/languages/prism/go";
import javascript from "react-syntax-highlighter/dist/esm/languages/prism/javascript";
import json from "react-syntax-highlighter/dist/esm/languages/prism/json";
import markdown from "react-syntax-highlighter/dist/esm/languages/prism/markdown";
import python from "react-syntax-highlighter/dist/esm/languages/prism/python";
import tsx from "react-syntax-highlighter/dist/esm/languages/prism/tsx";
import typescript from "react-syntax-highlighter/dist/esm/languages/prism/typescript";
import yaml from "react-syntax-highlighter/dist/esm/languages/prism/yaml";
import { oneLight } from "react-syntax-highlighter/dist/esm/styles/prism";
import remarkGfm from "remark-gfm";

SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("sh", bash);
SyntaxHighlighter.registerLanguage("shell", bash);
SyntaxHighlighter.registerLanguage("diff", diff);
SyntaxHighlighter.registerLanguage("go", go);
SyntaxHighlighter.registerLanguage("javascript", javascript);
SyntaxHighlighter.registerLanguage("js", javascript);
SyntaxHighlighter.registerLanguage("json", json);
SyntaxHighlighter.registerLanguage("markdown", markdown);
SyntaxHighlighter.registerLanguage("md", markdown);
SyntaxHighlighter.registerLanguage("python", python);
SyntaxHighlighter.registerLanguage("py", python);
SyntaxHighlighter.registerLanguage("typescript", typescript);
SyntaxHighlighter.registerLanguage("ts", typescript);
SyntaxHighlighter.registerLanguage("tsx", tsx);
SyntaxHighlighter.registerLanguage("yaml", yaml);
SyntaxHighlighter.registerLanguage("yml", yaml);

export function MarkdownBubbleContent({
  content,
  showCursor = false,
}: {
  content: string;
  showCursor?: boolean;
}) {
  return (
    <div className="space-y-2.5 text-[0.96rem] leading-7 text-[rgb(31,41,55)]">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => <h1 className="font-display text-[1.18rem] leading-8 text-[rgb(17,24,39)]">{children}</h1>,
          h2: ({ children }) => <h2 className="font-display text-[1.08rem] leading-7 text-[rgb(17,24,39)]">{children}</h2>,
          h3: ({ children }) => <h3 className="font-display text-[1rem] leading-7 text-[rgb(17,24,39)]">{children}</h3>,
          p: ({ children }) => <p className="whitespace-pre-wrap leading-7">{children}</p>,
          ul: ({ children }) => <ul className="list-disc space-y-1.5 pl-5">{children}</ul>,
          ol: ({ children }) => <ol className="list-decimal space-y-1.5 pl-5">{children}</ol>,
          li: ({ children }) => <li className="pl-1">{children}</li>,
          blockquote: ({ children }) => (
            <blockquote className="rounded-r-[1rem] border-l-[3px] border-[rgba(59,130,246,0.34)] bg-[rgba(248,250,255,0.9)] px-4 py-3 text-[rgba(55,65,81,0.92)]">
              <div className="space-y-1.5">{children}</div>
            </blockquote>
          ),
          hr: () => <div className="my-1 h-px bg-[rgba(226,231,239,0.96)]" />,
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noreferrer" className="text-[rgb(37,99,235)] underline decoration-[rgba(37,99,235,0.28)] underline-offset-4">
              {children}
            </a>
          ),
          strong: ({ children }) => <strong className="font-semibold text-[rgb(17,24,39)]">{children}</strong>,
          em: ({ children }) => <em className="italic">{children}</em>,
          code: ({ className, children, ...props }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) => {
            const inline = props.inline ?? !className?.includes("language-");
            const language = className?.replace("language-", "").trim();
            if (inline) {
              return (
                <code className="rounded-md border border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,255,0.98)] px-1.5 py-0.5 font-mono text-[0.82em] text-[rgb(37,99,235)]">
                  {children}
                </code>
              );
            }
            return (
              <div className="overflow-hidden rounded-[1.1rem] border border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,255,0.98)]">
                {language ? (
                  <div className="border-b border-[rgba(226,231,239,0.96)] px-4 py-2 font-mono text-[0.68rem] uppercase tracking-[0.18em] text-[rgba(107,114,128,0.76)]">
                    {language}
                  </div>
                ) : null}
                <SyntaxHighlighter
                  language={language || "text"}
                  style={oneLight}
                  customStyle={{
                    margin: 0,
                    padding: "0.9rem 1rem",
                    background: "rgba(248,250,255,0.98)",
                    fontSize: "0.83rem",
                    lineHeight: 1.7,
                    overflowX: "auto",
                  }}
                  codeTagProps={{
                    style: {
                      fontFamily:
                        'ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                    },
                  }}
                >
                  {String(children).replace(/\n$/, "")}
                </SyntaxHighlighter>
              </div>
            );
          },
        }}
      >
        {content}
      </ReactMarkdown>
      {showCursor && <span className="inline-block h-5 w-[2px] animate-pulse rounded-full bg-[rgb(37,99,235)] align-middle" />}
    </div>
  );
}
