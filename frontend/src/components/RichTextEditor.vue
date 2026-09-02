<template>
  <div class="rich-text-editor" :class="{ 'read-only': !editable, 'full-height': fullHeight }">
    <!-- Toolbar (only shown in edit mode, unless hideToolbar is true) -->
    <div v-if="editable && editor && !hideToolbar" class="editor-toolbar">
      <v-btn-group density="compact" variant="text">
        <v-btn
          size="small"
          :color="editor.isActive('bold') ? 'primary' : undefined"
          @click="editor.chain().focus().toggleBold().run()"
          title="Bold"
        >
          <v-icon size="18">mdi-format-bold</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive('italic') ? 'primary' : undefined"
          @click="editor.chain().focus().toggleItalic().run()"
          title="Italic"
        >
          <v-icon size="18">mdi-format-italic</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive('underline') ? 'primary' : undefined"
          @click="editor.chain().focus().toggleUnderline().run()"
          title="Underline"
        >
          <v-icon size="18">mdi-format-underline</v-icon>
        </v-btn>
      </v-btn-group>

      <v-divider vertical class="mx-2" />

      <v-btn-group density="compact" variant="text">
        <v-btn
          size="small"
          :color="editor.isActive('heading', { level: 1 }) ? 'primary' : undefined"
          @click="editor.chain().focus().toggleHeading({ level: 1 }).run()"
          title="Heading 1"
        >
          <v-icon size="18">mdi-format-header-1</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive('heading', { level: 2 }) ? 'primary' : undefined"
          @click="editor.chain().focus().toggleHeading({ level: 2 }).run()"
          title="Heading 2"
        >
          <v-icon size="18">mdi-format-header-2</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive('heading', { level: 3 }) ? 'primary' : undefined"
          @click="editor.chain().focus().toggleHeading({ level: 3 }).run()"
          title="Heading 3"
        >
          <v-icon size="18">mdi-format-header-3</v-icon>
        </v-btn>
      </v-btn-group>

      <v-divider vertical class="mx-2" />

      <v-btn-group density="compact" variant="text">
        <v-btn
          size="small"
          :color="editor.isActive('bulletList') ? 'primary' : undefined"
          @click="editor.chain().focus().toggleBulletList().run()"
          title="Bullet List"
        >
          <v-icon size="18">mdi-format-list-bulleted</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive('orderedList') ? 'primary' : undefined"
          @click="editor.chain().focus().toggleOrderedList().run()"
          title="Numbered List"
        >
          <v-icon size="18">mdi-format-list-numbered</v-icon>
        </v-btn>
      </v-btn-group>

      <v-divider vertical class="mx-2" />

      <v-btn-group density="compact" variant="text">
        <v-btn
          size="small"
          :color="editor.isActive({ textAlign: 'left' }) ? 'primary' : undefined"
          @click="editor.chain().focus().setTextAlign('left').run()"
          title="Align Left"
        >
          <v-icon size="18">mdi-format-align-left</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive({ textAlign: 'center' }) ? 'primary' : undefined"
          @click="editor.chain().focus().setTextAlign('center').run()"
          title="Align Center"
        >
          <v-icon size="18">mdi-format-align-center</v-icon>
        </v-btn>
        <v-btn
          size="small"
          :color="editor.isActive({ textAlign: 'right' }) ? 'primary' : undefined"
          @click="editor.chain().focus().setTextAlign('right').run()"
          title="Align Right"
        >
          <v-icon size="18">mdi-format-align-right</v-icon>
        </v-btn>
      </v-btn-group>

      <v-spacer />

      <v-btn
        size="small"
        variant="text"
        @click="editor.chain().focus().undo().run()"
        :disabled="!editor.can().undo()"
        title="Undo"
      >
        <v-icon size="18">mdi-undo</v-icon>
      </v-btn>
      <v-btn
        size="small"
        variant="text"
        @click="editor.chain().focus().redo().run()"
        :disabled="!editor.can().redo()"
        title="Redo"
      >
        <v-icon size="18">mdi-redo</v-icon>
      </v-btn>
    </div>

    <!-- Editor Content -->
    <editor-content :editor="editor" class="editor-content" />
  </div>
</template>

<script setup lang="ts">
import { watch, onBeforeUnmount } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'

const props = defineProps<{
  modelValue: string
  editable?: boolean
  fullHeight?: boolean  // When true, removes max-height constraint for use in scrollable containers
  hideToolbar?: boolean // When true, hides the built-in toolbar (for external toolbar placement)
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const editor = useEditor({
  content: props.modelValue,
  editable: props.editable ?? true,
  extensions: [
    StarterKit.configure({
      heading: {
        levels: [1, 2, 3],
      },
    }),
    Underline,
    TextAlign.configure({
      types: ['heading', 'paragraph'],
    }),
  ],
  onUpdate: () => {
    emit('update:modelValue', editor.value?.getHTML() || '')
  },
})

// Watch for external content changes
watch(() => props.modelValue, (newValue) => {
  if (editor.value && editor.value.getHTML() !== newValue) {
    editor.value.commands.setContent(newValue, { emitUpdate: false })
  }
})

// Watch for editable changes
watch(() => props.editable, (newValue) => {
  if (editor.value) {
    editor.value.setEditable(newValue ?? true)
  }
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})

// Expose editor instance for external toolbar control
defineExpose({
  editor
})
</script>

<style scoped>
.rich-text-editor {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  overflow: hidden;
  background: rgb(var(--v-theme-surface));
}

.editor-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  background: rgba(var(--v-theme-surface-variant), 0.3);
  flex-wrap: wrap;
  gap: 4px;
}

.editor-content {
  padding: 16px;
  min-height: 300px;
  max-height: 500px;
  overflow-y: auto;
}

/* When fullHeight is true, remove max-height to let parent container handle scrolling */
.rich-text-editor.full-height .editor-content {
  max-height: none;
  overflow-y: visible;
}

.editor-content :deep(.ProseMirror) {
  outline: none;
  min-height: 250px;
}

.editor-content :deep(.ProseMirror p) {
  margin: 0 0 0.75em 0;
  line-height: 1.6;
}

.editor-content :deep(.ProseMirror h1) {
  font-size: 1.75em;
  font-weight: 700;
  margin: 1em 0 0.5em 0;
  color: rgb(var(--v-theme-primary));
}

.editor-content :deep(.ProseMirror h2) {
  font-size: 1.4em;
  font-weight: 600;
  margin: 0.9em 0 0.4em 0;
  color: rgb(var(--v-theme-primary));
}

.editor-content :deep(.ProseMirror h3) {
  font-size: 1.2em;
  font-weight: 600;
  margin: 0.8em 0 0.3em 0;
  color: rgb(var(--v-theme-secondary));
}

.editor-content :deep(.ProseMirror ul),
.editor-content :deep(.ProseMirror ol) {
  padding-left: 1.5em;
  margin: 0.5em 0;
}

.editor-content :deep(.ProseMirror li) {
  margin: 0.25em 0;
}

.editor-content :deep(.ProseMirror strong) {
  font-weight: 700;
}

.editor-content :deep(.ProseMirror em) {
  font-style: italic;
}

.editor-content :deep(.ProseMirror u) {
  text-decoration: underline;
}

/* Read-only styling */
.read-only .editor-content {
  background: rgba(var(--v-theme-surface-variant), 0.3);
}

.read-only .editor-content :deep(.ProseMirror) {
  cursor: default;
}
</style>
