<template>
  <div class="er-diagram" ref="containerRef">
    <div class="er-diagram__toolbar">
      <el-button-group>
        <el-button size="small" @click="zoomIn"><el-icon><ZoomIn /></el-icon></el-button>
        <el-button size="small" @click="zoomOut"><el-icon><ZoomOut /></el-icon></el-button>
        <el-button size="small" @click="resetView"><el-icon><Refresh /></el-icon></el-button>
      </el-button-group>
      <el-button size="small" @click="autoLayout">{{ $t('db.teErAutoLayout') }}</el-button>
    </div>
    <div class="er-diagram__canvas" ref="canvasRef" @wheel="handleWheel">
      <svg :width="svgWidth" :height="svgHeight" :viewBox="viewBox">
        <!-- 关系线 -->
        <g class="er-diagram__relations">
          <line
            v-for="(rel, idx) in relations"
            :key="idx"
            :x1="rel.x1" :y1="rel.y1"
            :x2="rel.x2" :y2="rel.y2"
            :stroke="rel.color"
            stroke-width="2"
            :stroke-dasharray="rel.dashed ? '5,5' : 'none'"
          />
        </g>
        <!-- 表节点 -->
        <g class="er-diagram__tables">
          <g
            v-for="table in layoutTables"
            :key="table.name"
            :transform="`translate(${table.x}, ${table.y})`"
            class="er-diagram__table"
            :class="{ 'is-selected': selectedTable === table.name }"
            @click="handleTableClick(table.name)"
          >
            <rect width="200" :height="getTableHeight(table)" rx="4" fill="white" stroke="#dcdfe6" stroke-width="1" />
            <!-- 表头 -->
            <rect width="200" height="30" rx="4" fill="#409eff" />
            <text x="100" y="20" text-anchor="middle" fill="white" font-size="13" font-weight="600">{{ table.name }}</text>
            <!-- 列 -->
            <g v-for="(col, colIdx) in table.columns" :key="colIdx" :transform="`translate(0, ${30 + colIdx * 24})`">
              <rect x="0" width="200" height="24" :fill="colIdx % 2 === 0 ? '#fafafa' : 'white'" />
              <text x="8" :y="16" font-size="12" :fill="col.pri ? '#e6a23c' : '#303133'">
                {{ col.pri ? '🔑 ' : '' }}{{ col.name }}
              </text>
              <text x="192" :y="16" text-anchor="end" font-size="11" fill="#909399">{{ col.type }}</text>
            </g>
          </g>
        </g>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { ZoomIn, ZoomOut, Refresh } from '@element-plus/icons-vue';
import type { TableDefinition } from '../types/schema';

interface Props {
  tables: TableDefinition[];
}

const props = defineProps<Props>();
const emit = defineEmits<{ (e: 'table-click', tableName: string): void }>();

const containerRef = ref<HTMLElement>();
const canvasRef = ref<HTMLElement>();
const selectedTable = ref<string>('');
const scale = ref(1);
const offsetX = ref(0);
const offsetY = ref(0);

const svgWidth = computed(() => 2000);
const svgHeight = computed(() => 2000);
const viewBox = computed(() => `${offsetX.value} ${offsetY.value} ${2000 / scale.value} ${2000 / scale.value}`);

// 简单的网格布局
const layoutTables = computed(() => {
  const cols = Math.ceil(Math.sqrt(props.tables.length));
  return props.tables.map((table, idx) => ({
    ...table,
    x: 40 + (idx % cols) * 240,
    y: 40 + Math.floor(idx / cols) * 300,
  }));
});

// 计算关系线
const relations = computed(() => {
  const rels: Array<{ x1: number; y1: number; x2: number; y2: number; color: string; dashed: boolean }> = [];
  const tableMap = new Map(layoutTables.value.map(t => [t.name, t]));

  props.tables.forEach(table => {
    const source = tableMap.get(table.name);
    if (!source) return;

    table.constraints
      ?.filter(c => c.type === 'FOREIGN KEY' && c.referencedTable)
      .forEach(fk => {
        const target = tableMap.get(fk.referencedTable!);
        if (!target) return;
        rels.push({
          x1: source.x + 100,
          y1: source.y + getTableHeight(source),
          x2: target.x + 100,
          y2: target.y,
          color: '#409eff',
          dashed: false,
        });
      });
  });

  return rels;
});

function getTableHeight(table: TableDefinition): number {
  return 30 + table.columns.length * 24 + 10;
}

function handleTableClick(name: string) {
  selectedTable.value = name;
  emit('table-click', name);
}

function zoomIn() { scale.value = Math.min(scale.value * 1.2, 3); }
function zoomOut() { scale.value = Math.max(scale.value / 1.2, 0.3); }
function resetView() { scale.value = 1; offsetX.value = 0; offsetY.value = 0; }

function autoLayout() {
  // 简单的自动布局（网格布局已在computed中实现）
}

function handleWheel(e: WheelEvent) {
  e.preventDefault();
  if (e.deltaY < 0) { zoomIn(); } else { zoomOut(); }
}
</script>

<style scoped lang="scss">
.er-diagram {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  overflow: hidden;

  &__toolbar {
    display: flex;
    gap: 8px;
    padding: 8px 12px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  &__canvas {
    flex: 1;
    overflow: auto;
    background: #f5f7fa;
    cursor: grab;

    &:active { cursor: grabbing; }
  }

  &__table {
    cursor: pointer;
    transition: transform 0.2s;

    &:hover { filter: brightness(0.95); }
    &.is-selected rect:first-of-type { stroke: #409eff; stroke-width: 2; }
  }
}
</style>
