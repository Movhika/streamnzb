import React, { useState, useEffect, useMemo } from 'react'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Plus, X } from 'lucide-react'
import { DraggableList } from '@/components/ui/draggable-list'
import { Slider } from '@/components/ui/slider'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/**
 * PriorityList: value can be an array of keys (order) or legacy weight map.
 * onChange is always called with an array of keys (order). Top = highest priority (10 pts), bottom = 0.
 */
export function PriorityList({ items, value, onChange, title, description, showScoreHint = true }) {
  const isArrayValue = Array.isArray(value)

  // Build ordered list: from array value, or from legacy weight map
  const orderedItems = useMemo(() => {
    if (isArrayValue && value.length > 0) {
      const orderSet = new Set(value)
      const ordered = value.map(key => items.find(i => i.key === key)).filter(Boolean)
      const rest = items.filter(i => !orderSet.has(i.key))
      return [...ordered, ...rest]
    }
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return [...items].sort((a, b) => (value[b.key] || 0) - (value[a.key] || 0))
    }
    return [...items]
  }, [items, value, isArrayValue])

  const [localOrder, setLocalOrder] = useState(orderedItems)

  useEffect(() => {
    setLocalOrder(orderedItems)
  }, [orderedItems])

  const handleReorder = (newOrder) => {
    setLocalOrder(newOrder)
    onChange(newOrder.map(item => item.key))
  }

  const draggableItems = localOrder.map(item => ({
    id: item.key,
    label: item.label,
  }))

  return (
    <div className="space-y-3">
      <div>
        <h4 className="font-medium">{title}</h4>
        {description && <p className="text-sm text-muted-foreground mt-1">{description}</p>}
        {showScoreHint && (
          <p className="text-xs text-muted-foreground mt-0.5">
            Top = highest priority (10 pts), bottom = lowest (0 pts).
          </p>
        )}
      </div>
      <DraggableList
        items={draggableItems}
        onReorder={(newItems) => {
          const newOrder = newItems.map(dItem => localOrder.find(item => item.key === dItem.id)).filter(Boolean)
          handleReorder(newOrder)
        }}
      />
    </div>
  )
}

/**
 * PrioritySubsetList: user chooses a subset of options and orders them (add/remove).
 * Value = array of keys in order. Empty by default; user adds what they care about.
 */
function SortableSubsetItem({ id, label, onRemove }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 p-3 bg-background border rounded-md hover:bg-accent/50 transition-colors"
    >
      <div
        {...attributes}
        {...listeners}
        className="cursor-grab active:cursor-grabbing text-muted-foreground hover:text-foreground"
      >
        <GripVertical className="h-5 w-5" />
      </div>
      <div className="flex-1">{label}</div>
      <button
        type="button"
        className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-muted"
        onClick={() => onRemove(id)}
        aria-label={`Remove ${label}`}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

export function PrioritySubsetList({ allItems, value = [], onChange, title, description, showScoreHint = true }) {
  const order = Array.isArray(value) ? value : []
  const itemsInOrder = order.map(key => allItems.find(i => i.key === key)).filter(Boolean)
  const availableToAdd = allItems.filter(i => !order.includes(i.key))

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const handleDragEnd = (event) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIndex = itemsInOrder.findIndex(item => item.key === active.id)
    const newIndex = itemsInOrder.findIndex(item => item.key === over.id)
    if (oldIndex === -1 || newIndex === -1) return
    const reordered = arrayMove(itemsInOrder, oldIndex, newIndex)
    onChange(reordered.map(i => i.key))
  }

  const add = (key) => {
    if (key && !order.includes(key)) onChange([...order, key])
  }

  const remove = (key) => {
    onChange(order.filter(k => k !== key))
  }

  return (
    <div className="space-y-3">
      <div>
        <h4 className="font-medium">{title}</h4>
        {description && <p className="text-sm text-muted-foreground mt-1">{description}</p>}
        {showScoreHint && (
          <p className="text-xs text-muted-foreground mt-0.5">
            Top = highest priority (10 pts), bottom = lowest (0 pts). Add only what you care about.
          </p>
        )}
      </div>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={itemsInOrder.map(i => i.key)} strategy={verticalListSortingStrategy}>
          <div className="space-y-2">
            {itemsInOrder.map((item) => (
              <SortableSubsetItem
                key={item.key}
                id={item.key}
                label={item.label}
                onRemove={remove}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
      {availableToAdd.length > 0 && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm" className="mt-2 gap-1">
              <Plus className="h-3.5 w-3.5" />
              Add
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="max-h-[14rem] overflow-y-auto">
            {availableToAdd.map((item) => (
              <button
                key={item.key}
                type="button"
                className="w-full cursor-pointer rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
                onClick={() => add(item.key)}
              >
                {item.label}
              </button>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      {order.length === 0 && (
        <p className="text-sm text-muted-foreground mt-1">Add items above to set priority. Leave empty to ignore this category.</p>
      )}
    </div>
  )
}

export function MultiplierSlider({ label, value = 1.0, onChange, min = 0, max = 2, step = 0.1, description }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="font-medium">{label}</h4>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
        <span className="text-sm font-mono bg-muted px-2 py-1 rounded">{value.toFixed(1)}</span>
      </div>
      <Slider
        value={[value]}
        onValueChange={(values) => onChange(values[0])}
        min={min}
        max={max}
        step={step}
        className="w-full"
      />
    </div>
  )
}
