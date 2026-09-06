/** Super-admin how-to body. HTML is sanitized on the ledger; images must be /howto/ files. */
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'

export function HowToEditor({
  html,
  plates,
  onChange,
}: {
  html: string
  plates: string[]
  onChange: (html: string) => void
}) {
  const editor = useEditor({
    immediatelyRender: true,
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] },
        codeBlock: false,
        code: false,
        link: false,
      }),
      Image.configure({ allowBase64: false }),
      Placeholder.configure({
        placeholder: 'Write the bay card. Do not invent pins on a module connector. DLC 16 / 4 / 5 only for the under-dash port.',
      }),
    ],
    content: html || '<p></p>',
    onUpdate: ({ editor: ed }) => onChange(ed.getHTML()),
  })

  if (!editor) return <p className="font-mono text-xs text-steel">Editor loading…</p>

  function insertPlate(file: string) {
    if (!file || !editor) return
    editor.chain().focus().setImage({ src: `/howto/${file}`, alt: file }).run()
  }

  return (
    <div className="howto-editor">
      <div className="howto-editor-bar" role="toolbar" aria-label="How-to formatting">
        <button type="button" className={editor.isActive('bold') ? 'is-on' : ''} onClick={() => editor.chain().focus().toggleBold().run()}>B</button>
        <button type="button" className={editor.isActive('italic') ? 'is-on' : ''} onClick={() => editor.chain().focus().toggleItalic().run()}><em>I</em></button>
        <button type="button" className={editor.isActive('heading', { level: 3 }) ? 'is-on' : ''} onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}>H3</button>
        <button type="button" className={editor.isActive('bulletList') ? 'is-on' : ''} onClick={() => editor.chain().focus().toggleBulletList().run()}>UL</button>
        <button type="button" className={editor.isActive('orderedList') ? 'is-on' : ''} onClick={() => editor.chain().focus().toggleOrderedList().run()}>OL</button>
        <button type="button" onClick={() => editor.chain().focus().setHorizontalRule().run()}>HR</button>
        <label className="howto-editor-plate">
          <span>PLATE</span>
          <select defaultValue="" onChange={(e) => { insertPlate(e.target.value); e.target.value = '' }}>
            <option value="">/howto/…</option>
            {plates.map((f) => (
              <option key={f} value={f}>{f}</option>
            ))}
          </select>
        </label>
      </div>
      <EditorContent editor={editor} />
    </div>
  )
}
