/**
 * 资源对象 Kind 元数据（前端 fallback 表）。
 *
 * 字段顺序与 BFF 返回的 ResourceKindMeta 保持一致；当 BFF 返回 description 缺失时
 * 使用本表兜底。Kind 名称必须与 Go 端 `internal/domain/resource.AllKinds` 保持一致，
 * 新增 Kind 需同步更新两侧。
 */
export interface KindMetaFallback {
  kind: string
  collection: string
  idField: string
  description: string
}

export const KIND_METAS: KindMetaFallback[] = [
  { kind: 'Person',          collection: 'Persons',         idField: 'PersonID',         description: '人员' },
  { kind: 'Face',            collection: 'Faces',           idField: 'FaceID',           description: '人脸' },
  { kind: 'MotorVehicle',    collection: 'MotorVehicles',   idField: 'MotorVehicleID',   description: '机动车' },
  { kind: 'NonMotorVehicle', collection: 'NonMotorVehicles', idField: 'NonMotorVehicleID', description: '非机动车' },
  { kind: 'Thing',           collection: 'Things',          idField: 'ThingID',          description: '物品' },
  { kind: 'Scene',           collection: 'Scenes',          idField: 'SceneID',          description: '场景' },
  { kind: 'VideoSlice',      collection: 'VideoSlices',     idField: 'VideoSliceID',     description: '视频片段' },
  { kind: 'Image',           collection: 'Images',          idField: 'ImageID',          description: '图像' },
  { kind: 'File',            collection: 'Files',           idField: 'FileID',           description: '文件' },
  { kind: 'Case',            collection: 'Cases',           idField: 'CaseID',           description: '案件' },
  { kind: 'VideoLabel',      collection: 'VideoLabels',     idField: 'VideoLabelID',     description: '视频标签' },
  { kind: 'AnalysisRule',    collection: 'AnalysisRules',   idField: 'AnalysisRuleID',   description: '分析规则' },
]

/** 从信封里抽出 <Kind>Object 数组。容忍 key 命名差异（Person/Persons 等）。 */
export function extractObjects(envelope: Record<string, unknown>, kind: string): Record<string, unknown>[] {
  if (!envelope || typeof envelope !== 'object') return []
  const listKey = kind + 'List'
  const objKey = kind + 'Object'
  const list = envelope[listKey] as Record<string, unknown> | undefined
  if (list && Array.isArray(list[objKey])) return list[objKey] as Record<string, unknown>[]
  // 容错：有些实现把 objKey 放在 envelope 顶层
  if (Array.isArray(envelope[objKey])) return envelope[objKey] as Record<string, unknown>[]
  return []
}

/** 提取对象主键值（如 PersonID / MotorVehicleID）。 */
export function pickID(obj: Record<string, unknown>, idField: string): string {
  const v = obj?.[idField]
  return v === undefined || v === null ? '' : String(v)
}