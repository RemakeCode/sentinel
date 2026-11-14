import { ConfirmModal, findModuleExport, Focusable, ScrollPanel, showModal } from '@decky/ui';
import type { ComponentType, ReactNode } from 'react';
import { Markdown } from '@tanstack/markdown/react';
import type { UpdateRelease } from '@/pages/settings/updates/update-status';

//language=css
const styles = `
  .sentinel-patch-notes {
    max-height: 100px;
    overflow: hidden;
    padding: 16px;
    background: rgba(37, 40, 46, 0.5);
    cursor: pointer;
    overflow-wrap: anywhere;
  }
  .sentinel-patch-notes.gpfocus { outline: 2px solid white; outline-offset: 2px; }
  .sentinel-patch-notes-title { font-weight: bold; text-transform: uppercase; margin-bottom: 8px; }
  .sentinel-patch-notes-description { font-size: 14px; color: #b8bcbf; margin-bottom: 8px; }
  .sentinel-patch-notes-body { overflow-wrap: anywhere; }
  .sentinel-patch-notes-body > :first-child { margin-top: 0; }
  .sentinel-patch-notes-full { height: 60vh; overflow-y: auto; overflow-wrap: anywhere; }
  .DialogContent.sentinel-patch-notes-modal { width: 85vw; max-width: 1100px; }
  .sentinel-patch-notes-modal .DialogFooter { display: flex; justify-content: flex-end; }
  .sentinel-patch-notes-modal .DialogFooter > .DialogButton { flex: 0 0 auto; width: 160px; max-width: 100%; }
`;

interface TextScrollPanelProps {
  children: ReactNode;
  className: string;
  scrollStepPercent: number;
  scrollBehavior: 'smooth';
}

// Steam uses this ScrollPanel variant for continuous gamepad scrolling in EULAs.
// The regular @decky/ui ScrollPanel only scrolls when navigating between children.
const TextScrollPanel = (findModuleExport((value: unknown) => {
  const render = (value as { render?: unknown } | null)?.render;
  if (typeof render !== 'function') return false;
  const source = render.toString();
  return source.includes('scrollStepPercent') && source.includes('onGamepadDirection');
}) || ScrollPanel) as ComponentType<TextScrollPanelProps>;

function PatchNotesModal({ release, closeModal }: { release: UpdateRelease; closeModal: () => void }) {
  return (
    <ConfirmModal
      className='sentinel-patch-notes-modal'
      bAllowFullSize
      strTitle={`Sentinel ${release.version} — Release notes`}
      strOKButtonText='Close'
      bAlertDialog
      onOK={closeModal}
      onCancel={closeModal}
      onEscKeypress={closeModal}
    >
      <style>{styles}</style>
      <TextScrollPanel className='sentinel-patch-notes-full' scrollStepPercent={40} scrollBehavior='smooth'>
        <Focusable autoFocus noFocusRing {...{ focusable: true }}>
          <Markdown allowHtml>{release.notes || 'No release notes provided.'}</Markdown>
        </Focusable>
      </TextScrollPanel>
    </ConfirmModal>
  );
}

export function ReleaseNotes({ release }: { release: UpdateRelease }) {
  const openNotes = () => {
    let modal: ReturnType<typeof showModal>;
    modal = showModal(<PatchNotesModal release={release} closeModal={() => modal.Close()} />);
  };

  return (
    <>
      <style>{styles}</style>
      <Focusable
        className='sentinel-patch-notes'
        onActivate={openNotes}
        onOptionsButton={openNotes}
        onOptionsActionDescription='Patch notes'
      >
        <div className='sentinel-patch-notes-title'>Release {release.version}</div>
        <div className='sentinel-patch-notes-description'>
          These release notes may include both desktop and Decky changes.
        </div>
        <div className='sentinel-patch-notes-body'>
          <Markdown allowHtml>{release.notes || 'No release notes provided.'}</Markdown>
        </div>
      </Focusable>
    </>
  );
}
