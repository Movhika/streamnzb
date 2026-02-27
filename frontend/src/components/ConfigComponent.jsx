import React from 'react'
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
  horizontalListSortingStrategy,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Plus, X } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { FormField, FormItem, FormControl } from '@/components/ui/form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  { key: 'size', label: 'Size' },
]

/** Draggable pill for priority / preferred list */
function SortablePill({ id, label, onRemove }) {
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
      className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-md border bg-muted/60 hover:bg-muted text-sm shrink-0"
    >
      <span {...attributes} {...listeners} className="cursor-grab active:cursor-grabbing text-muted-foreground">
        <GripVertical className="h-3.5 w-3.5" />
      </span>
      <span>{label}</span>
      {onRemove && (
        <button
          type="button"
          className="rounded p-0.5 hover:bg-muted-foreground/20 text-muted-foreground hover:text-foreground"
          onClick={() => onRemove(id)}
          aria-label={`Remove ${label}`}
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  )
}

/** One category block: Preferred (order) + Excluded. Options shared, no duplicates. */
function CategoryBlock({
  title,
  orderValue,
  onOrderChange,
  avoidValue,
  onAvoidChange,
  items,
}) {
  const orderList = Array.isArray(orderValue) ? orderValue : []
  const avoidList = Array.isArray(avoidValue) ? avoidValue : []
  const usedSet = new Set([...orderList, ...avoidList])
  const available = items.filter((i) => !usedSet.has(i.key))
  const orderWithLabels = orderList
    .map((key) => ({ id: key, label: items.find((i) => i.key === key)?.label ?? key }))
    .filter((i) => i.label)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const handleOrderDragEnd = (event) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIdx = orderList.indexOf(active.id)
    const newIdx = orderList.indexOf(over.id)
    if (oldIdx === -1 || newIdx === -1) return
    onOrderChange(arrayMove([...orderList], oldIdx, newIdx))
  }

  const addToPreferred = (key) => {
    if (!key || orderList.includes(key)) return
    onOrderChange([...orderList, key])
    if (avoidList.includes(key)) onAvoidChange(avoidList.filter((k) => k !== key))
  }
  const removeFromPreferred = (key) => onOrderChange(orderList.filter((k) => k !== key))
  const addToExcluded = (key) => {
    if (!key || avoidList.includes(key)) return
    onAvoidChange([...avoidList, key])
    if (orderList.includes(key)) onOrderChange(orderList.filter((k) => k !== key))
  }
  const removeFromExcluded = (key) => onAvoidChange(avoidList.filter((k) => k !== key))

  return (
    <div className="space-y-3 py-3 border-b border-border last:border-b-0">
      <h4 className="text-sm font-medium text-foreground">{title}</h4>
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <p className="text-xs text-muted-foreground mb-1.5">Preferred (left = first)</p>
          <div className="flex flex-wrap gap-2 items-center min-h-[2.5rem] p-2 rounded-md border bg-background">
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleOrderDragEnd}>
              <SortableContext items={orderList} strategy={horizontalListSortingStrategy}>
                {orderWithLabels.map((item) => (
                  <SortablePill
                    key={item.id}
                    id={item.id}
                    label={item.label}
                    onRemove={removeFromPreferred}
                  />
                ))}
              </SortableContext>
            </DndContext>
            {available.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="ghost" size="sm" className="h-8 gap-1 text-muted-foreground">
                    <Plus className="h-3.5 w-3.5" />
                    Add
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="max-h-[12rem] overflow-y-auto">
                  {available.map((item) => (
                    <button
                      key={item.key}
                      type="button"
                      className="w-full cursor-pointer rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
                      onClick={() => addToPreferred(item.key)}
                    >
                      {item.label}
                    </button>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>
        <div>
          <p className="text-xs text-muted-foreground mb-1.5">Excluded</p>
          <div className="flex flex-wrap gap-1.5 min-h-[2.5rem] p-2 rounded-md border border-destructive/30 bg-destructive/5">
            {avoidList.map((key) => {
              const label = items.find((i) => i.key === key)?.label ?? key
              return (
                <span
                  key={key}
                  className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs bg-destructive/20 text-destructive border border-destructive/30"
                >
                  {label}
                  <button
                    type="button"
                    className="rounded p-0.5 hover:bg-destructive/30"
                    onClick={() => removeFromExcluded(key)}
                    aria-label={`Remove ${label} from excluded`}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              )
            })}
            {available.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="ghost" size="sm" className="h-7 gap-1 text-destructive/80 hover:text-destructive">
                    <Plus className="h-3 w-3" />
                    Add
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="max-h-[12rem] overflow-y-auto">
                  {available.map((item) => (
                    <button
                      key={item.key}
                      type="button"
                      className="w-full cursor-pointer rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
                      onClick={() => addToExcluded(item.key)}
                    >
                      {item.label}
                    </button>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>
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
          Sort by: drag to set which category matters most (first = highest priority). Per category: set Preferred order and Excluded; each option can only be in one place.
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
            return (
              <FormField
                key={cat.key}
                control={control}
                name={getFieldName(cat.orderField)}
                render={({ field: orderField }) => (
                  <FormField
                    control={control}
                    name={getFieldName(cat.avoidField)}
                    render={({ field: avoidField }) => (
                      <FormItem className="border-0 p-0 m-0">
                        <FormControl>
                          <div className="px-4">
                            <CategoryBlock
                              title={cat.label}
                              orderValue={orderField.value}
                              onOrderChange={orderField.onChange}
                              avoidValue={avoidField.value}
                              onAvoidChange={avoidField.onChange}
                              items={cat.items}
                            />
                          </div>
                        </FormControl>
                      </FormItem>
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
