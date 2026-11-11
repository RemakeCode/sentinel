import './release-notes.scss';
import type { FC } from 'react';
import { Markdown, type MarkdownComponents } from '@tanstack/markdown/react';
import { Browser } from '@wailsio/runtime';

const markdownComponents = {
  a: ({ href, children }) => (
    <a href={href} onClick={(event) => {
      event.preventDefault();
      if (href) void Browser.OpenURL(href);
    }}>{children}</a>
  )
} satisfies MarkdownComponents;

const ReleaseNotes: FC<{ notes: string; className?: string }> = ({ notes, className = '' }) => (
  <div className={`release-notes ${className}`}>
    <Markdown allowHtml components={markdownComponents}>{notes || 'No release notes provided.'}</Markdown>
  </div>
);

export default ReleaseNotes;
