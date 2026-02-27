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

/** Single sortable pill for priority column (horizontal drag) */
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

/** Option row: priority (order) + filter out. Options can only be in one place (no duplicates). */
function OptionGridRow({
  rowIndex,
  categoryLabel,
  orderValue,
  onOrderChange,
  avoidValue,
  onAvoidChange,
  orderItems,
  onRemove,
}) {
  const orderList = Array.isArray(orderValue) ? orderValue : []
  const avoidList = Array.isArray(avoidValue) ? avoidValue : []
  const usedSet = new Set([...orderList, ...avoidList])
  const available = orderItems.filter((i) => !usedSet.has(i.key))
  const orderWithLabels = orderList
    .map((key) => ({ id: key, label: orderItems.find((i) => i.key === key)?.label ?? key }))
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

  const addToOrder = (key) => {
    if (!key || orderList.includes(key)) return
    onOrderChange([...orderList, key])
    if (avoidList.includes(key)) onAvoidChange(avoidList.filter((k) => k !== key))
  }
  const removeFromOrder = (key) => onOrderChange(orderList.filter((k) => k !== key))
  const addToAvoid = (key) => {
    if (!key || avoidList.includes(key)) return
    onAvoidChange([...avoidList, key])
    if (orderList.includes(key)) onOrderChange(orderList.filter((k) => k !== key))
  }
  const removeFromAvoid = (key) => onAvoidChange(avoidList.filter((k) => k !== key))

  return (
    <div
      className={cn(
        'grid gap-3 items-start py-3 border-b border-border',
        'grid-cols-[minmax(0,1fr)_2fr_minmax(0,1.2fr)] md:grid-cols-[140px_1fr_180px]'
      )}
    >
      <div className="pt-2 flex items-start justify-between gap-2">
        <div>
          <span className="text-sm font-medium text-foreground">{categoryLabel}</span>
          <span className="block text-xs text-muted-foreground mt-0.5">Priority {rowIndex}</span>
        </div>
        {onRemove && (
          <button
            type="button"
            className="rounded p-1 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
            onClick={onRemove}
            aria-label={`Remove ${categoryLabel}`}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="min-w-0">
        <p className="text-xs text-muted-foreground mb-1.5">Priority (left = first)</p>
        <div className="flex flex-wrap gap-2 items-center min-h-[2.5rem] p-2 rounded-md border bg-background">
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleOrderDragEnd}>
            <SortableContext items={orderList} strategy={horizontalListSortingStrategy}>
              {orderWithLabels.map((item) => (
                <SortablePill
                  key={item.id}
                  id={item.id}
                  label={item.label}
                  onRemove={removeFromOrder}
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
                    onClick={() => addToOrder(item.key)}
                  >
                    {item.label}
                  </button>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      <div className="min-w-0">
        <p className="text-xs text-muted-foreground mb-1.5">Filter out</p>
        <div className="flex flex-wrap gap-1.5 min-h-[2.5rem] p-2 rounded-md border border-destructive/30 bg-destructive/5">
          {avoidList.map((key) => {
            const label = orderItems.find((i) => i.key === key)?.label ?? key
            return (
              <span
                key={key}
                className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs bg-destructive/20 text-destructive border border-destructive/30"
              >
                {label}
                <button
                  type="button"
                  className="rounded p-0.5 hover:bg-destructive/30"
                  onClick={() => removeFromAvoid(key)}
                  aria-label={`Remove ${label} from filter out`}
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
                    onClick={() => addToAvoid(item.key)}
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
  )
}

/** Size row: min/max GB only (no priority list). 0 = no limit for max. */
function SizeGridRow({ minSizeGb, maxSizeGb, onMinChange, onMaxChange }) {
  const minVal = minSizeGb != null && Number(minSizeGb) >= 0 ? Number(minSizeGb) : 0
  const maxVal = maxSizeGb != null && Number(maxSizeGb) > 0 ? Number(maxSizeGb) : ''
  return (
    <div
      className={cn(
        'grid gap-3 items-start py-3 border-b border-border',
        'grid-cols-[minmax(0,1fr)_2fr_minmax(0,1.2fr)] md:grid-cols-[140px_1fr_180px]'
      )}
    >
      <div className="pt-2">
        <span className="text-sm font-medium text-foreground">Size</span>
        <span className="block text-xs text-muted-foreground mt-0.5">Filter by file size range</span>
      </div>
      <div className="min-w-0 flex flex-wrap items-center gap-3 col-span-2">
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

/**
 * Priority grid: default rows (Resolution, Size) + addable category rows.
 * Resolution and optional categories have priority order + filter out (shared options, no duplicates).
 * Size is a min/max range only.
 */
export function PriorityGrid({
  control,
  fieldPrefix = '',
  resolutionRow,
  sizeRow,
  addableCategories,
  addedCategoryKeys,
  onAddedCategoryKeysChange,
}) {
  const getFieldName = (field) => (fieldPrefix ? `${fieldPrefix}.${field}` : field)
  const availableToAdd = addableCategories.filter((c) => !addedCategoryKeys.includes(c.key))

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Priority grid</CardTitle>
        <CardDescription>
          Resolution and size by default. Add categories (source quality, codec, etc.) as needed. Each option can be in priority order or filter out, not both.
        </CardDescription>
      </CardHeader>
      <CardContent className="p-4">
        <div className="rounded-lg border bg-muted/20">
          <div
            className={cn(
              'grid gap-3 px-4 py-2 border-b border-border bg-muted/30 text-xs font-medium text-muted-foreground',
              'grid-cols-[minmax(0,1fr)_2fr_minmax(0,1.2fr)] md:grid-cols-[140px_1fr_180px]'
            )}
          >
            <div>Category</div>
            <div>Priority (left = first)</div>
            <div>Filter out</div>
          </div>
          <div className="divide-y divide-border">
            {/* Resolution row (always first) */}
            {resolutionRow && (
              <FormField
                control={control}
                name={getFieldName(resolutionRow.orderField)}
                render={({ field: orderField }) => (
                  <FormField
                    control={control}
                    name={getFieldName(resolutionRow.avoidField)}
                    render={({ field: avoidField }) => (
                      <FormItem className="border-0 p-0 m-0">
                        <FormControl>
                          <div className="px-4">
                            <OptionGridRow
                              rowIndex={1}
                              categoryLabel={resolutionRow.label}
                              orderValue={orderField.value}
                              onOrderChange={orderField.onChange}
                              avoidValue={avoidField.value}
                              onAvoidChange={avoidField.onChange}
                              orderItems={resolutionRow.items}
                            />
                          </div>
                        </FormControl>
                      </FormItem>
                    )}
                  />
                )}
              />
            )}

            {/* Size row */}
            {sizeRow && (
              <FormField
                control={control}
                name={getFieldName(sizeRow.minField)}
                render={({ field: minField }) => (
                  <FormField
                    control={control}
                    name={getFieldName(sizeRow.maxField)}
                    render={({ field: maxField }) => (
                      <FormItem className="border-0 p-0 m-0">
                        <FormControl>
                          <div className="px-4">
                            <SizeGridRow
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
            )}

            {/* Added category rows */}
            {addedCategoryKeys.map((key, idx) => {
              const cat = addableCategories.find((c) => c.key === key)
              if (!cat) return null
              return (
                <FormField
                  key={key}
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
                              <OptionGridRow
                                rowIndex={2 + idx + 1}
                                categoryLabel={cat.label}
                                orderValue={orderField.value}
                                onOrderChange={orderField.onChange}
                                avoidValue={avoidField.value}
                                onAvoidChange={avoidField.onChange}
                                orderItems={cat.items}
                                onRemove={() => {
                                  onAddedCategoryKeysChange(addedCategoryKeys.filter((k) => k !== key))
                                }}
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

          {/* Add category */}
          {availableToAdd.length > 0 && (
            <div className="px-4 py-3 border-t border-border">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="outline" size="sm" className="gap-2">
                    <Plus className="h-4 w-4" />
                    Add category
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="max-h-[14rem] overflow-y-auto">
                  {availableToAdd.map((cat) => (
                    <button
                      key={cat.key}
                      type="button"
                      className="w-full cursor-pointer rounded-sm px-3 py-2 text-left text-sm hover:bg-accent"
                      onClick={() => onAddedCategoryKeysChange([...addedCategoryKeys, cat.key])}
                    >
                      {cat.label}
                    </button>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
