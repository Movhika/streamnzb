import React from 'react'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  useDraggable,
  useDroppable,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  horizontalListSortingStrategy,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Plus, X } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { FormField, FormItem, FormControl } from '@/components/ui/form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Slider } from '@/components/ui/slider'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

const SORT_CRITERIA_OPTIONS = [
  { key: 'resolution', label: 'Resolution' },
  { key: 'quality', label: 'Source quality' },
  { key: 'codec', label: 'Codec' },
  { key: 'visual_tag', label: 'Visual tags' },
  { key: 'audio', label: 'Audio' },
  { key: 'channels', label: 'Channels' },
  { key: 'bit_depth', label: 'Bit depth' },
  { key: 'container', label: 'Container' },
  { key: 'languages', label: 'Languages' },
  { key: 'group', label: 'Release group' },
  { key: 'edition', label: 'Edition' },
  { key: 'network', label: 'Network' },
  { key: 'region', label: 'Region' },
  { key: 'three_d', label: '3D' },
  { key: 'size', label: 'Size' },
  { key: 'keywords', label: 'Keywords' },
  { key: 'regex', label: 'Regex patterns' },
  { key: 'availnzb', label: 'AvailNZB status' },
]

const COL_SEP = '\u001f' // unit separator: columnId + COL_SEP + key (key can contain dashes)

function parseDragId(dragId) {
  const i = dragId.indexOf(COL_SEP)
  if (i < 0) return { column: null, key: dragId }
  return { column: dragId.slice(0, i), key: dragId.slice(i + 1) }
}

function makeDragId(column, key) {
  return `${column}${COL_SEP}${key}`
}

/** Draggable chip for Included or Excluded column (no internal reorder). */
function DraggableChip({ column, itemKey, label, onRemove, className, buttonClass }) {
  const id = makeDragId(column, itemKey)
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id, data: { column, key: itemKey } })
  const handleRemove = (e) => { e.stopPropagation(); onRemove(itemKey) }
  return (
    <span
      ref={setNodeRef}
      className={cn('inline-flex items-center gap-1 px-2 py-1 rounded text-xs border shrink-0 cursor-grab active:cursor-grabbing', className, isDragging && 'opacity-60')}
      {...attributes}
      {...listeners}
    >
      {label}
      <button
        type="button"
        className={cn('rounded p-0.5 hover:opacity-80', buttonClass)}
        onClick={handleRemove}
        aria-label={`Remove ${label}`}
      >
        <X className="h-3 w-3" />
      </button>
    </span>
  )
}

/** Draggable + sortable pill for Preferred list; id must be makeDragId('preferred', key). */
function SortablePill({ id, itemKey, label, onRemove }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  }
  return (
    <div
      ref={setNodeRef}
      style={style}
      className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-md border bg-muted/60 hover:bg-muted text-sm shrink-0 cursor-grab active:cursor-grabbing"
    >
      <span {...attributes} {...listeners} className="text-muted-foreground flex">
        <GripVertical className="h-3.5 w-3.5" />
      </span>
      <span>{label}</span>
      {onRemove && (
        <button
          type="button"
          className="rounded p-0.5 hover:bg-muted-foreground/20 text-muted-foreground hover:text-foreground"
          onClick={(e) => { e.stopPropagation(); onRemove(itemKey) }}
          aria-label={`Remove ${label}`}
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  )
}

function AddDropdown({ available, onAdd, align = "start", variant = "ghost", className }) {
  if (available.length === 0) return null
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant={variant} size="sm" className={cn("h-7 gap-1", className)}>
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="max-h-[12rem] overflow-y-auto">
        {available.map((item) => (
          <button
            key={item.key}
            type="button"
            className="w-full cursor-pointer rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
            onClick={() => onAdd(item.key)}
          >
            {item.label}
          </button>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

/** Category block: Included (bypass) + Preferred (order) + Excluded. Drag between columns or reorder within Preferred. */
function CategoryBlock({
  title,
  includedValue,
  onIncludedChange,
  orderValue,
  onOrderChange,
  excludedValue,
  onExcludedChange,
  items,
}) {
  const includedList = Array.isArray(includedValue) ? includedValue : []
  const orderList = Array.isArray(orderValue) ? orderValue : []
  const excludedList = Array.isArray(excludedValue) ? excludedValue : []
  const usedSet = new Set([...includedList, ...orderList, ...excludedList])
  const available = items.filter((i) => !usedSet.has(i.key))
  const orderWithLabels = orderList
    .map((key) => ({ key, label: items.find((i) => i.key === key)?.label ?? key }))
    .filter((i) => i.label)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const removeFromAll = (key) => {
    if (includedList.includes(key)) onIncludedChange(includedList.filter((k) => k !== key))
    if (orderList.includes(key)) onOrderChange(orderList.filter((k) => k !== key))
    if (excludedList.includes(key)) onExcludedChange(excludedList.filter((k) => k !== key))
  }

  const addToIncluded = (key) => {
    if (!key || includedList.includes(key)) return
    removeFromAll(key)
    onIncludedChange([...includedList.filter((k) => k !== key), key])
  }
  const removeFromIncluded = (key) => onIncludedChange(includedList.filter((k) => k !== key))

  const addToPreferred = (key, beforeKey = null) => {
    removeFromAll(key)
    if (beforeKey != null && orderList.includes(beforeKey)) {
      const without = orderList.filter((k) => k !== key)
      const idx = without.indexOf(beforeKey)
      const insertIdx = idx < 0 ? without.length : idx
      const next = [...without.slice(0, insertIdx), key, ...without.slice(insertIdx)]
      onOrderChange(next)
    } else {
      onOrderChange([...orderList.filter((k) => k !== key), key])
    }
  }
  const removeFromPreferred = (key) => onOrderChange(orderList.filter((k) => k !== key))

  const addToExcluded = (key) => {
    if (!key || excludedList.includes(key)) return
    removeFromAll(key)
    onExcludedChange([...excludedList.filter((k) => k !== key), key])
  }
  const removeFromExcluded = (key) => onExcludedChange(excludedList.filter((k) => k !== key))
  const preferredIds = orderList.map((k) => makeDragId('preferred', k))

  const handleDragEnd = (event) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const activeStr = String(active.id)
    const overStr = String(over.id)
    const { column: sourceCol, key } = parseDragId(activeStr)
    if (!key) return

    if (overStr.startsWith('zone' + COL_SEP)) {
      const targetCol = overStr.slice(5) // 'zone\x1f' -> rest is column name
      if (targetCol === 'included') addToIncluded(key)
      else if (targetCol === 'excluded') addToExcluded(key)
      else if (targetCol === 'preferred') addToPreferred(key)
      return
    }

    const overParsed = parseDragId(overStr)
    if (overParsed.column === 'preferred' && overParsed.key) {
      if (sourceCol === 'preferred') {
        const oldIdx = orderList.indexOf(key)
        const newIdx = orderList.indexOf(overParsed.key)
        if (oldIdx !== -1 && newIdx !== -1) onOrderChange(arrayMove([...orderList], oldIdx, newIdx))
      } else {
        addToPreferred(key, overParsed.key)
      }
    }
  }

  return (
    <div className="space-y-3 py-3 border-b border-border last:border-b-0">
      <h4 className="text-sm font-medium text-foreground">{title}</h4>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={preferredIds} strategy={horizontalListSortingStrategy}>
          <div className="grid gap-4 sm:grid-cols-3">
            {/* Included (bypass) */}
            <DroppableZone column="included" className="flex flex-wrap gap-1.5 min-h-[2.5rem] p-2 rounded-md border border-blue-500/30 bg-blue-500/5">
              {includedList.map((k) => (
                <DraggableChip
                  key={k}
                  column="included"
                  itemKey={k}
                  label={items.find((i) => i.key === k)?.label ?? k}
                  onRemove={removeFromIncluded}
                  className="bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30"
                  buttonClass="hover:bg-blue-500/30"
                />
              ))}
              <AddDropdown available={available} onAdd={addToIncluded} className="text-blue-500/80 hover:text-blue-500" />
            </DroppableZone>
            {/* Preferred (sort order) */}
            <DroppableZone column="preferred" className="flex flex-wrap gap-2 items-center min-h-[2.5rem] p-2 rounded-md border bg-background">
              {orderWithLabels.map((item) => (
                <SortablePill
                  key={item.key}
                  id={makeDragId('preferred', item.key)}
                  itemKey={item.key}
                  label={item.label}
                  onRemove={removeFromPreferred}
                />
              ))}
              <AddDropdown available={available} onAdd={addToPreferred} className="text-muted-foreground" />
            </DroppableZone>
            {/* Excluded */}
            <DroppableZone column="excluded" className="flex flex-wrap gap-1.5 min-h-[2.5rem] p-2 rounded-md border border-destructive/30 bg-destructive/5">
              {excludedList.map((k) => (
                <DraggableChip
                  key={k}
                  column="excluded"
                  itemKey={k}
                  label={items.find((i) => i.key === k)?.label ?? k}
                  onRemove={removeFromExcluded}
                  className="bg-destructive/20 text-destructive border-destructive/30"
                  buttonClass="hover:bg-destructive/30"
                />
              ))}
              <AddDropdown available={available} onAdd={addToExcluded} align="end" className="text-destructive/80 hover:text-destructive" />
            </DroppableZone>
          </div>
        </SortableContext>
      </DndContext>
    </div>
  )
}

function DroppableZone({ column, className, children }) {
  const { setNodeRef, isOver } = useDroppable({ id: `zone${COL_SEP}${column}` })
  const zoneStyles = {
    included: 'border border-blue-500/30 bg-blue-500/5',
    preferred: 'border bg-background',
    excluded: 'border border-destructive/30 bg-destructive/5',
  }
  return (
    <div>
      <p className={cn('text-xs mb-1.5', column === 'included' && 'text-blue-500', column === 'preferred' && 'text-muted-foreground', column === 'excluded' && 'text-destructive')}>
        {column === 'included' && 'Included (bypass)'}
        {column === 'preferred' && 'Preferred (left = first)'}
        {column === 'excluded' && 'Excluded'}
      </p>
      <div ref={setNodeRef} className={cn(className, zoneStyles[column], isOver && 'ring-2 ring-primary ring-offset-2')}>
        {children}
      </div>
    </div>
  )
}

/** Free-text category block: same 3-column drag-drop but values are user-entered (no fixed list). firstColumnLabel defaults to "Included (bypass)". */
function FreeTextCategoryBlock({
  title,
  firstColumnLabel = 'Included (bypass)',
  includedValue,
  onIncludedChange,
  orderValue,
  onOrderChange,
  excludedValue,
  onExcludedChange,
}) {
  const includedList = Array.isArray(includedValue) ? includedValue : []
  const orderList = Array.isArray(orderValue) ? orderValue : []
  const excludedList = Array.isArray(excludedValue) ? excludedValue : []
  const allKeys = [...new Set([...includedList, ...orderList, ...excludedList])]
  const items = allKeys.map((s) => ({ key: s, label: s }))

  const addToColumn = (column, raw) => {
    const parts = raw.split(/[\s,]+/).map((s) => s.trim()).filter(Boolean)
    if (parts.length === 0) return
    const existing = column === 'included' ? includedList : column === 'preferred' ? orderList : excludedList
    const seen = new Set(existing)
    const toAdd = parts.filter((p) => !seen.has(p))
    if (toAdd.length === 0) return
    if (column === 'included') onIncludedChange([...includedList, ...toAdd])
    else if (column === 'preferred') onOrderChange([...orderList, ...toAdd])
    else onExcludedChange([...excludedList, ...toAdd])
  }

  const removeFromAll = (key) => {
    onIncludedChange(includedList.filter((k) => k !== key))
    onOrderChange(orderList.filter((k) => k !== key))
    onExcludedChange(excludedList.filter((k) => k !== key))
  }
  const addToIncluded = (key) => { removeFromAll(key); onIncludedChange([...includedList.filter((k) => k !== key), key]) }
  const addToPreferred = (key, beforeKey = null) => {
    removeFromAll(key)
    if (beforeKey != null && orderList.includes(beforeKey)) {
      const without = orderList.filter((k) => k !== key)
      const idx = without.indexOf(beforeKey)
      const next = [...without.slice(0, idx < 0 ? without.length : idx), key, ...without.slice(idx < 0 ? without.length : idx)]
      onOrderChange(next)
    } else onOrderChange([...orderList.filter((k) => k !== key), key])
  }
  const addToExcluded = (key) => { removeFromAll(key); onExcludedChange([...excludedList.filter((k) => k !== key), key]) }
  const preferredIds = orderList.map((k) => makeDragId('preferred', k))
  const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }))

  const handleDragEnd = (event) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const { column: sourceCol, key } = parseDragId(String(active.id))
    if (!key) return
    const overStr = String(over.id)
    if (overStr.startsWith('zone' + COL_SEP)) {
      const targetCol = overStr.slice(5)
      if (targetCol === 'included') addToIncluded(key)
      else if (targetCol === 'excluded') addToExcluded(key)
      else if (targetCol === 'preferred') addToPreferred(key)
      return
    }
    const overParsed = parseDragId(overStr)
    if (overParsed.column === 'preferred' && overParsed.key) {
      if (sourceCol === 'preferred') {
        const oldIdx = orderList.indexOf(key)
        const newIdx = orderList.indexOf(overParsed.key)
        if (oldIdx !== -1 && newIdx !== -1) onOrderChange(arrayMove([...orderList], oldIdx, newIdx))
      } else addToPreferred(key, overParsed.key)
    }
  }

  const orderWithLabels = orderList.map((k) => ({ key: k, label: k }))
  const removeFromIncluded = (k) => onIncludedChange(includedList.filter((x) => x !== k))
  const removeFromPreferred = (k) => onOrderChange(orderList.filter((x) => x !== k))
  const removeFromExcluded = (k) => onExcludedChange(excludedList.filter((x) => x !== k))

  return (
    <div className="space-y-3 py-3 border-b border-border last:border-b-0">
      <h4 className="text-sm font-medium text-foreground">{title}</h4>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={preferredIds} strategy={horizontalListSortingStrategy}>
          <div className="grid gap-4 sm:grid-cols-3">
            <FreeTextColumn
              column="included"
              label={firstColumnLabel}
              list={includedList}
              onAdd={(raw) => addToColumn('included', raw)}
              onRemove={removeFromIncluded}
              onDrop={(key) => addToIncluded(key)}
              items={items}
            />
            <FreeTextColumn
              column="preferred"
              label="Preferred (left = first)"
              list={orderList}
              orderWithLabels={orderWithLabels}
              onAdd={(raw) => addToColumn('preferred', raw)}
              onRemove={removeFromPreferred}
              onDrop={(key) => addToPreferred(key)}
              items={items}
              sortable
            />
            <FreeTextColumn
              column="excluded"
              label="Excluded"
              list={excludedList}
              onAdd={(raw) => addToColumn('excluded', raw)}
              onRemove={removeFromExcluded}
              onDrop={(key) => addToExcluded(key)}
              items={items}
            />
          </div>
        </SortableContext>
      </DndContext>
    </div>
  )
}

function FreeTextColumn({ column, label, list, onAdd, onRemove, onDrop, items, orderWithLabels, sortable }) {
  const [inputVal, setInputVal] = React.useState('')
  const { setNodeRef, isOver } = useDroppable({ id: `zone${COL_SEP}${column}` })
  const zoneStyles = { included: 'border-blue-500/30 bg-blue-500/5', preferred: 'bg-background', excluded: 'border-destructive/30 bg-destructive/5' }
  const labelClass = column === 'included' ? 'text-blue-500' : column === 'preferred' ? 'text-muted-foreground' : 'text-destructive'
  const handleKeyDown = (e) => { if (e.key === 'Enter') { e.preventDefault(); onAdd(inputVal); setInputVal('') } }
  const handlePaste = (e) => {
    const pasted = (e.clipboardData || window.clipboardData)?.getData('text')
    if (pasted && /[\s,]/.test(pasted)) { e.preventDefault(); onAdd(pasted); setInputVal('') }
  }
  return (
    <div>
      <p className={cn('text-xs mb-1.5', labelClass)}>{label}</p>
      <div ref={setNodeRef} className={cn('flex flex-wrap gap-1.5 min-h-[2.5rem] p-2 rounded-md border', zoneStyles[column], isOver && 'ring-2 ring-primary ring-offset-2')}>
        {column === 'preferred' && orderWithLabels?.map((item) => (
          <SortablePill key={item.key} id={makeDragId('preferred', item.key)} itemKey={item.key} label={item.label} onRemove={onRemove} />
        ))}
        {column !== 'preferred' && list.map((k) => (
          <DraggableChip
            key={k}
            column={column}
            itemKey={k}
            label={k}
            onRemove={onRemove}
            className={column === 'included' ? 'bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30' : 'bg-destructive/20 text-destructive border-destructive/30'}
            buttonClass={column === 'included' ? 'hover:bg-blue-500/30' : 'hover:bg-destructive/30'}
          />
        ))}
        <Input
          placeholder="Add…"
          className="h-7 w-24 text-xs flex-shrink-0"
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
        />
      </div>
    </div>
  )
}



/** Size block: min/max GB only */
function SizeBlock({ minSizeGb, maxSizeGb, onMinChange, onMaxChange }) {
  const minVal = minSizeGb != null && Number(minSizeGb) >= 0 ? Number(minSizeGb) : 0
  const maxVal = maxSizeGb != null && Number(maxSizeGb) > 0 ? Number(maxSizeGb) : ''
  return (
    <div className="space-y-3 py-3 border-b border-border last:border-b-0">
      <h4 className="text-sm font-medium text-foreground">Size</h4>
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted-foreground whitespace-nowrap">Min (GB)</label>
          <Input
            type="number"
            min={0}
            step={0.5}
            placeholder="0"
            className="w-20"
            value={minVal}
            onChange={(e) => {
              const v = parseFloat(e.target.value)
              onMinChange(Number.isFinite(v) && v >= 0 ? v : 0)
            }}
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted-foreground whitespace-nowrap">Max (GB)</label>
          <Input
            type="number"
            min={0}
            step={0.5}
            placeholder="No max"
            className="w-20"
            value={maxVal}
            onChange={(e) => {
              if (e.target.value === '') {
                onMaxChange(0)
                return
              }
              const v = parseFloat(e.target.value)
              onMaxChange(Number.isFinite(v) && v >= 0 ? v : 0)
            }}
          />
        </div>
        <span className="text-xs text-muted-foreground">Leave max empty for no limit.</span>
      </div>
    </div>
  )
}

/** Sortable criterion item for the "Sort by" list */
function SortCriterionItem({ id, label }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  }
  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 px-3 py-2 rounded-md border bg-background hover:bg-accent/50"
    >
      <span {...attributes} {...listeners} className="cursor-grab active:cursor-grabbing text-muted-foreground">
        <GripVertical className="h-4 w-4" />
      </span>
      <span className="text-sm font-medium">{label}</span>
    </div>
  )
}

export function ConfigComponent({ control, fieldPrefix = '', criteriaOrderValue, onCriteriaOrderChange, categories }) {
  const getFieldName = (field) => (fieldPrefix ? `${fieldPrefix}.${field}` : field)
  const orderList = Array.isArray(criteriaOrderValue) ? criteriaOrderValue : []
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const handleCriteriaDragEnd = (event) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIdx = orderList.indexOf(active.id)
    const newIdx = orderList.indexOf(over.id)
    if (oldIdx === -1 || newIdx === -1) return
    onCriteriaOrderChange(arrayMove([...orderList], oldIdx, newIdx))
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Filters &amp; sorting</CardTitle>
        <CardDescription>
          Sort by: drag to set which category matters most (first = highest priority). Per category: set{' '}
          <span className="text-blue-500 font-medium">Included</span> (bypass all filters),{' '}
          <span className="font-medium">Preferred</span> order, and{' '}
          <span className="text-destructive font-medium">Excluded</span>; each option can only be in one place.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Sort by (first = highest priority)</h4>
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleCriteriaDragEnd}>
            <SortableContext items={orderList} strategy={verticalListSortingStrategy}>
              <div className="flex flex-col gap-2">
                {orderList.map((key) => {
                  const opt = SORT_CRITERIA_OPTIONS.find((o) => o.key === key)
                  if (!opt) return null
                  return <SortCriterionItem key={opt.key} id={opt.key} label={opt.label} />
                })}
              </div>
            </SortableContext>
          </DndContext>
        </div>

        <div className="rounded-lg border bg-muted/20 divide-y divide-border">
          {categories.map((cat) => {
            if (cat.type === 'size') {
              return (
                <FormField
                  key="size"
                  control={control}
                  name={getFieldName(cat.minField)}
                  render={({ field: minField }) => (
                    <FormField
                      control={control}
                      name={getFieldName(cat.maxField)}
                      render={({ field: maxField }) => (
                        <FormItem className="border-0 p-0 m-0">
                          <FormControl>
                            <div className="px-4">
                              <SizeBlock
                                minSizeGb={minField.value}
                                maxSizeGb={maxField.value}
                                onMinChange={minField.onChange}
                                onMaxChange={maxField.onChange}
                              />
                            </div>
                          </FormControl>
                        </FormItem>
                      )}
                    />
                  )}
                />
              )
            }
            if (cat.type === 'freeText') {
              return (
                <FormField
                  key={cat.key}
                  control={control}
                  name={getFieldName(cat.includedField)}
                  render={({ field: includedField }) => (
                    <FormField
                      control={control}
                      name={getFieldName(cat.orderField)}
                      render={({ field: orderField }) => (
                        <FormField
                          control={control}
                          name={getFieldName(cat.excludedField)}
                          render={({ field: excludedField }) => (
                            <FormItem className="border-0 p-0 m-0">
                              <FormControl>
                                <div className="px-4">
                                  <FreeTextCategoryBlock
                                    title={cat.label}
                                    firstColumnLabel={cat.firstColumnLabel}
                                    includedValue={includedField.value}
                                    onIncludedChange={includedField.onChange}
                                    orderValue={orderField.value}
                                    onOrderChange={orderField.onChange}
                                    excludedValue={excludedField.value}
                                    onExcludedChange={excludedField.onChange}
                                  />
                                </div>
                              </FormControl>
                            </FormItem>
                          )}
                        />
                      )}
                    />
                  )}
                />
              )
            }

            return (
              <FormField
                key={cat.key}
                control={control}
                name={getFieldName(cat.includedField)}
                render={({ field: includedField }) => (
                  <FormField
                    control={control}
                    name={getFieldName(cat.orderField)}
                    render={({ field: orderField }) => (
                      <FormField
                        control={control}
                        name={getFieldName(cat.excludedField)}
                        render={({ field: excludedField }) => (
                          <FormItem className="border-0 p-0 m-0">
                            <FormControl>
                              <div className="px-4">
                                <CategoryBlock
                                  title={cat.label}
                                  includedValue={includedField.value}
                                  onIncludedChange={includedField.onChange}
                                  orderValue={orderField.value}
                                  onOrderChange={orderField.onChange}
                                  excludedValue={excludedField.value}
                                  onExcludedChange={excludedField.onChange}
                                  items={cat.items}
                                />
                              </div>
                            </FormControl>
                          </FormItem>
                        )}
                      />
                    )}
                  />
                )}
              />
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
