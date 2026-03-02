export interface UserOrg {
  id: string;
  name: string;
  is_primary: boolean;
  position?: string;
}

export interface User {
  id: string;
  email: string;
  phone_country_code?: string;
  phone_number?: string;
  name: string;
  role: 'admin' | 'operator' | 'viewer';
  permissions: string;
  status: 'active' | 'disabled';
  avatar?: string;
  last_login?: string;
  created_at: string;
  updated_at: string;
  // 组织机构信息
  organizations?: UserOrg[];
  primary_org_name?: string;
}

export interface PortGeofence {
  id: string;
  code: string;
  name: string;
  name_cn: string;
  country: string;
  country_cn: string;
  geofence_type: string;
  center_lat: number;
  center_lng: number;
  radius: number;
  polygon_points?: string;
  color: string;
  is_active: boolean;
}

export type CustomerType = 'sender' | 'receiver';

export interface Customer {
  id: string;
  org_id?: string;
  type: CustomerType;
  name: string;
  phone: string;
  company?: string;
  address?: string;
  city?: string;
  country?: string;
  latitude?: number;
  longitude?: number;
  created_at: string;
  updated_at?: string;
}

export interface Device {
  id: string;
  name: string;
  type: 'container' | 'truck' | 'vessel' | 'other';
  provider: 'kuaihuoyun' | 'g7' | 'sinoiov' | 'yilian';
  status: 'online' | 'offline';
  battery: number;
  latitude?: number;
  longitude?: number;
  external_device_id?: string;
  speed: number;
  direction: number;
  temperature?: number;
  humidity?: number;
  locate_type?: number;
  last_update: string;
  created_at: string;
  // 绑定运单信息
  binding_status?: 'bound' | 'unbound';
  bound_transport_type?: string;
  bound_cargo_name?: string;
  bound_shipment_id?: string;
  // 组织归属
  org_id?: string;
  org_name?: string;
}

export interface Shipment {
  id: string;
  device_id?: string;
  org_id?: string;                        // 所属组织ID
  org_name?: string;                      // 所属组织名称
  transport_type?: string;              // 运输类型: sea/air/land/multimodal
  cargo_name?: string;                  // 货物名称
  origin: string;
  destination: string;
  origin_lat?: number;
  origin_lng?: number;
  dest_lat?: number;
  dest_lng?: number;
  origin_radius?: number;           // 发货地围栏半径(米)
  dest_radius?: number;             // 目的地围栏半径(米)
  auto_status_enabled?: boolean;    // 启用自动状态切换
  current_milestone?: string;       // 当前里程碑
  status: 'pending' | 'in_transit' | 'delivered' | 'cancelled';
  progress: number;
  departure_time?: string;
  eta?: string;
  device_bound_at?: string;
  left_origin_at?: string;
  arrived_dest_at?: string;         // 到达目的地时间
  status_updated_at?: string;
  created_at: string;
  device?: Device;
  unbound_device_id?: string; // 解绑后的设备ID记录
  total_duration?: string;    // 总耗时（从开始运输到签收/当前）

  // 关键单证 (标准跨境型)
  bill_of_lading?: string;          // 提单号 MBL/HBL/AWB
  container_no?: string;            // 箱号/车牌号
  seal_no?: string;                 // 封条号 (Phase 1新增)
  container_type?: string;          // 柜型

  // 发货/收货信息
  sender_name?: string;
  sender_phone?: string;
  origin_address?: string;
  receiver_name?: string;
  receiver_phone?: string;
  dest_address?: string;

  // 货物详情
  cargo_type?: string;              // 货物类型
  transport_mode?: string;          // 运输模式 (FCL/LCL)

  // 详细路由 - ETD/ATD/ATA
  etd?: string;                     // 预计出发时间
  atd?: string;                     // 实际出发时间
  ata?: string;                     // 实际到达时间 (Phase 1新增)

  // 船务信息 (Phase 1新增)
  vessel_name?: string;             // 船名
  voyage_no?: string;               // 航次
  carrier?: string;                 // 船司/航司

  // 订单关联 (Phase 1新增)
  po_numbers?: string;              // PO单号(可多个,逗号分隔)
  sku_ids?: string;                 // SKU ID(可多个)
  fba_shipment_id?: string;         // FBA发货编号

  // 货物量纲
  pieces?: number;                  // 件数
  weight?: number;                  // 重量 (kg)
  volume?: number;                  // 体积 (m³)

  // 费用记录 (Phase 1新增)
  freight_cost?: number;            // 运费 USD
  surcharges?: number;              // 附加费 USD
  customs_fee?: number;             // 关税 USD
  other_cost?: number;              // 其他费用 USD
  total_cost?: number;              // 总费用 USD

  // IoT 预警阈值
  max_temperature?: number;
  min_temperature?: number;
  max_humidity?: number;
  max_shock?: number;
  max_tilt?: number;

  // 关务与末端作业状态
  customs_status?: string;          // pending, examination, cleared, hold
  customs_hold_since?: string;
  free_time_expiration?: string;
  appointment_status?: string;      // none, scheduled, failed, completed
}

export interface LocationHistory {
  id: number;
  device_id: string;
  shipment_id?: string;
  latitude: number;
  longitude: number;
  speed: number;
  direction: number;
  temperature?: number;
  humidity?: number;
  locate_type?: number;
  timestamp: string;
}

export interface TrackPoint {
  device: string;
  speed: number;
  direction: number;
  locateTime: number;
  longitude: number;
  latitude: number;
  locateType: number;
  runStatus: number;
  temperature?: number;
  humidity?: number;
}

export interface Alert {
  id: string;
  device_id?: string;
  shipment_id?: string;
  category: 'physical' | 'node' | 'operation' | 'carrier' | 'system'; // 预警类别 (Phase 2新增carrier)
  type: string;
  severity: 'info' | 'warning' | 'critical';
  title: string;
  message?: string;
  location?: string;
  status: 'pending' | 'resolved';
  created_at: string;
  resolved_at?: string;
  device?: Device;
  shipment?: Shipment;
}

export interface DashboardStats {
  devices: {
    total: number;
    online: number;
    offline: number;
  };
  shipments: {
    total: number;
    pending: number;
    in_transit: number;
    delivered: number;
  };
  alerts: {
    pending: number;
    critical: number;
  };
}

export interface LoginRequest {
  email: string;
  phone_country_code?: string;
  phone_number?: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// 运单操作日志
export interface ShipmentLog {
  id: number;
  shipment_id: string;
  action: string;          // created, updated, status_changed, device_bound, device_unbound, etc.
  field?: string;          // 变更字段
  old_value?: string;      // 旧值
  new_value?: string;      // 新值
  description: string;     // 操作描述
  operator_id?: string;    // 操作人ID
  operator_ip?: string;    // 操作人IP
  created_at: string;
}

// 设备绑定历史
export interface DeviceBinding {
  id: number;
  shipment_id: string;
  device_id: string;
  bound_at: string;
  unbound_at?: string;
  unbound_reason?: string;  // replaced, completed, manual
  created_at: string;
}

// 组织机构类型
export type OrganizationType = 'group' | 'company' | 'branch' | 'dept' | 'team';

// 组织机构
export interface Organization {
  id: string;
  name: string;
  code: string;
  parent_id?: string;
  type: OrganizationType;
  level: number;
  path: string;
  sort: number;
  status: 'active' | 'disabled';
  leader_id?: string;
  leader_name?: string;
  description?: string;
  user_count?: number;
  device_count?: number;
  created_at: string;
  updated_at: string;
  children?: Organization[];
}

// 组织树节点
export interface OrganizationTreeNode {
  id: string;
  name: string;
  code: string;
  parent_id?: string;
  type: OrganizationType;
  level: number;
  sort: number;
  status: string;
  leader_name?: string;
  user_count: number;
  device_count: number;
  children?: OrganizationTreeNode[];
}

// 用户组织关联
export interface UserOrganization {
  id: number;
  user_id: string;
  organization_id: string;
  organization_name?: string;
  organization_code?: string;
  is_primary: boolean;
  position?: string;
  joined_at: string;
  user?: User;
}

// 创建组织请求
export interface CreateOrganizationRequest {
  name: string;
  code: string;
  parent_id?: string;
  type?: OrganizationType;
  sort?: number;
  leader_id?: string;
  description?: string;
}

// 更新组织请求
export interface UpdateOrganizationRequest {
  name?: string;
  code?: string;
  type?: OrganizationType;
  sort?: number;
  status?: 'active' | 'disabled';
  leader_id?: string;
  description?: string;
}

// 移动组织请求
export interface MoveOrganizationRequest {
  parent_id?: string;
  sort?: number;
}

// 添加用户到组织请求
export interface AddUserToOrgRequest {
  user_id: string;
  is_primary?: boolean;
  position?: string;
}

// 更新用户组织信息请求
export interface UpdateUserOrgRequest {
  is_primary?: boolean;
  position?: string;
}

// ============ Phase 2: 船司追踪类型 ============

// 船司追踪事件
export interface CarrierTrack {
  id: number;
  shipment_id: string;
  bill_of_lading: string;
  event_code: string;      // GATE_OUT, LOADED, VESSEL_DEPARTURE...
  event_name: string;
  location: string;
  lo_code?: string;        // UN/LOCODE
  latitude?: number;
  longitude?: number;
  vessel_name?: string;
  voyage_no?: string;
  carrier?: string;
  event_time: string;
  eta_update?: string;
  is_actual: boolean;
  source: string;          // mock, vizion, p44...
  synced_at: string;
  created_at: string;
}

// 运单里程碑 (统一IoT和船司事件)
export interface ShipmentMilestone {
  id: number;
  shipment_id: string;
  code: string;           // 标准事件代码
  name: string;           // 中文名称
  sequence: number;       // 排序
  status: 'planned' | 'actual' | 'skipped';
  planned_time?: string;
  actual_time?: string;
  source: 'iot' | 'carrier' | 'manual' | 'geofence';
  location?: string;
  lo_code?: string;
  carrier_track_id?: number;
  device_track_id?: number;
  remark?: string;
  created_at: string;
}

// 船务事件代码常量
export const CarrierEventCodes = {
  GATE_OUT: '提柜出场',
  LOADED: '装船完成',
  VESSEL_DEPARTURE: '船舶离港',
  TRANSSHIPMENT: '中转换船',
  VESSEL_ARRIVAL: '船舶到港',
  DISCHARGE: '卸船完成',
  CUSTOMS_HOLD: '海关查验',
  CUSTOMS_RELEASE: '海关放行',
  GATE_IN: '还柜进场',
  DELIVERY: '签收完成',
} as const;

// ============ Phase 3: 多方协同类型 ============

// 物流环节类型
export type LogisticsStage = 'first_mile' | 'origin_port' | 'main_leg' | 'dest_port' | 'last_mile';

// 物流环节名称
export const LogisticsStageNames: Record<LogisticsStage, string> = {
  first_mile: '前程运输',
  origin_port: '起运港',
  main_leg: '干线运输',
  dest_port: '目的港',
  last_mile: '末端配送',
};

// ============ Phase 6: 运输环节流转类型 ============

// 环节代码（主环节7个 + 硬件触发6个子事件）
export type StageCode =
  // 主运输环节7个（UI展示用）
  | 'pre_transit'        // 前程运输
  | 'origin_port'        // 起运港
  | 'main_line'          // 干线运输
  | 'transit_port'       // 中转港（可选）
  | 'dest_port'          // 目的港
  | 'last_mile'          // 末端配送
  | 'delivered'          // 签收
  // 硬件自动触发的子事件（6个，记录日志用）
  | 'origin_arrival'     // 起运港到港
  | 'origin_departure'   // 起运港离港
  | 'transit_arrival'    // 中转港到港
  | 'transit_departure'  // 中转港离港
  | 'dest_arrival'       // 目的港到港
  | 'dest_departure'     // 目的港离港
  // 兼容旧版代码
  | 'pickup'             // 旧: 揽收 -> 映射到 pre_transit
  | 'first_mile'         // 旧: 首公里 -> 映射到 pre_transit
  | 'main_carriage'      // 旧: 干线运输
  | 'delivery';          // 旧: 末端派送 -> 映射到 last_mile

// 环节状态
export type StageStatus = 'pending' | 'in_progress' | 'completed' | 'skipped';

// 触发方式
export type TriggerType = 'manual' | 'geofence' | 'api';

// 运输环节
export interface ShipmentStage {
  id: string;
  shipment_id: string;
  stage_code: StageCode;
  stage_name: string;
  stage_icon: string;
  stage_order: number;
  status: StageStatus;
  partner_id?: string;
  partner_name?: string;

  // 关键数据
  vehicle_plate?: string;   // 前程运输：拖车车牌
  vessel_name?: string;     // 起运港/干线：船名
  voyage_no?: string;       // 航次
  carrier?: string;         // 船司/航司
  port_code?: string;       // 港口代码

  // 港口坐标（用于地图渲染）
  port_lat?: number;
  port_lng?: number;
  port_name?: string;

  // 时间节点
  planned_start?: string;
  planned_end?: string;
  actual_start?: string;
  actual_end?: string;

  // 费用
  cost_name?: string;  // 费用名称（如：干线运输费）
  cost: number;
  currency: string;

  // 触发信息
  trigger_type?: TriggerType;
  trigger_note?: string;
}

// 获取环节列表响应
export interface ShipmentStagesResponse {
  stages: ShipmentStage[];
  current_stage: StageCode;
  total_cost: number;
}

// 更新环节请求
export interface UpdateStageRequest {
  status?: StageStatus;
  partner_id?: string;
  partner_name?: string;
  vehicle_plate?: string;
  vessel_name?: string;
  voyage_no?: string;
  carrier?: string;
  actual_start?: string;
  actual_end?: string;
  cost?: number;
  currency?: string;
  trigger_note?: string;
}

// 环节代码到中文名称映射
export const StageCodeNames: Partial<Record<StageCode, string>> = {
  // 主运输环节7个（UI展示）
  pre_transit: '前程运输',
  origin_port: '起运港',
  main_line: '干线运输',
  transit_port: '中转港',
  dest_port: '目的港',
  last_mile: '末端配送',
  delivered: '签收',
  // 硬件自动触发子事件6个（日志用）
  origin_arrival: '起运港到港',
  origin_departure: '起运港离港',
  transit_arrival: '中转港到港',
  transit_departure: '中转港离港',
  dest_arrival: '目的港到港',
  dest_departure: '目的港离港',
  // 兼容旧版代码
  pickup: '前程运输',
  first_mile: '前程运输',
  main_carriage: '干线运输',
  delivery: '末端配送',
};

// 环节图标
export const StageCodeIcons: Partial<Record<StageCode, string>> = {
  // 主运输环节7个
  pre_transit: '🚚',
  origin_port: '🚢',
  main_line: '🌊',
  transit_port: '⚓',
  dest_port: '🏁',
  last_mile: '🚚',
  delivered: '✅',
  // 硬件触发子事件
  origin_arrival: '🚢⬇️',
  origin_departure: '🚢⬆️',
  transit_arrival: '⚓⬇️',
  transit_departure: '⚓⬆️',
  dest_arrival: '🏁⬇️',
  dest_departure: '🏁⬆️',
  // 兼容旧版
  pickup: '🚚',
  first_mile: '🚛',
  main_carriage: '🌊',
  delivery: '🚚',
}

// 环节状态名称
export const StageStatusNames: Record<StageStatus, string> = {
  pending: '待开始',
  in_progress: '进行中',
  completed: '已完成',
  skipped: '已跳过',
};

// 合作伙伴类型 (扩展到16种)
export type PartnerType =
  // 前程运输
  | 'drayage_origin'    // 首程拖车
  | 'consolidator'      // 集运/拼箱
  | 'inspector'         // 质检机构
  // 起运港
  | 'booking_agent'     // 订舱代理
  | 'customs_export'    // 出口报关行
  | 'terminal'          // 码头/堆场
  // 干线运输
  | 'vocc'              // 船公司
  | 'nvocc'             // 无船承运人
  | 'airline'           // 航空公司
  // 目的港
  | 'customs_import'    // 进口清关行
  | 'drayage_dest'      // 目的港拖车
  | 'chassis_provider'  // 底盘车队
  | 'bonded_warehouse'  // 保税仓
  // 末端配送
  | 'overseas_3pl'      // 海外仓
  | 'courier'           // 快递公司
  | 'platform_warehouse' // 电商平台仓
  // 兼容旧类型
  | 'forwarder'         // 货代(通用)
  | 'broker'            // 报关行(通用)
  | 'trucker'           // 拖车行(通用)
  | 'warehouse';        // 仓库(通用)

// 合作伙伴
export interface Partner {
  id: string;
  name: string;
  code: string;
  type: PartnerType;
  stage?: LogisticsStage;
  sub_type?: string;
  contact_name: string;
  phone: string;
  email: string;
  phone_country_code?: string;
  phone_number?: string;
  address?: string;
  country?: string;
  region?: string;
  service_ports?: string;
  service_routes?: string;
  rating: number;
  total_shipments: number;
  on_time_rate?: number;
  certifications?: string;
  api_config?: string;
  service_capabilities?: string;
  contract_info?: string;
  payment_terms?: string;
  insurance_coverage?: number;
  status: 'active' | 'inactive';
  owner_org_id: string;
  created_at: string;
}

// 合作伙伴类型名称 (按环节分组)
export const PartnerTypeNames: Record<PartnerType, string> = {
  // 前程运输
  drayage_origin: '首程拖车',
  consolidator: '集运/拼箱',
  inspector: '质检机构',
  // 起运港
  booking_agent: '订舱代理',
  customs_export: '出口报关行',
  terminal: '码头/堆场',
  // 干线运输
  vocc: '船公司',
  nvocc: '无船承运人',
  airline: '航空公司',
  // 目的港
  customs_import: '进口清关行',
  drayage_dest: '目的港拖车',
  chassis_provider: '底盘车队',
  bonded_warehouse: '保税仓',
  // 末端配送
  overseas_3pl: '海外仓',
  courier: '快递公司',
  platform_warehouse: '电商平台仓',
  // 兼容旧类型
  forwarder: '货代',
  broker: '报关行',
  trucker: '拖车行',
  warehouse: '仓库',
};

// 按物流环节分组的合作伙伴类型
export const PartnerTypesByStage: Record<LogisticsStage, PartnerType[]> = {
  first_mile: ['drayage_origin', 'consolidator', 'inspector'],
  origin_port: ['booking_agent', 'customs_export', 'terminal'],
  main_leg: ['vocc', 'nvocc', 'airline'],
  dest_port: ['customs_import', 'drayage_dest', 'chassis_provider', 'bonded_warehouse'],
  last_mile: ['overseas_3pl', 'courier', 'platform_warehouse'],
};

// 协作状态
export type CollaborationStatus = 'invited' | 'accepted' | 'in_progress' | 'completed' | 'cancelled';

// 运单协作记录
export interface ShipmentCollaboration {
  id: number;
  shipment_id: string;
  partner_id: string;
  partner_name?: string;
  partner_type?: PartnerType;
  role: PartnerType;
  status: CollaborationStatus;
  assigned_at: string;
  accepted_at?: string;
  completed_at?: string;
  task_desc?: string;
  remarks?: string;
  assigned_by: string;
  assigned_by_name?: string;
}

// 单据类型
export type DocumentType = 'bl' | 'invoice' | 'packing_list' | 'customs_dec' | 'co' | 'insurance' | 'contract' | 'other';

// 单据状态
export type DocumentStatus = 'pending' | 'approved' | 'rejected';

// 运单单据
export interface ShipmentDocument {
  id: number;
  shipment_id: string;
  partner_id?: string;
  partner_name?: string;
  uploader_id: string;
  uploader_name?: string;
  doc_type: DocumentType;
  doc_type_name: string;
  doc_name: string;
  file_name: string;
  file_size: number;
  mime_type: string;
  status: DocumentStatus;
  reviewer_id?: string;
  reviewed_at?: string;
  remarks?: string;
  uploaded_at: string;
  download_url?: string;
}

// 单据类型名称
export const DocumentTypeNames: Record<DocumentType, string> = {
  bl: '提单',
  invoice: '商业发票',
  packing_list: '装箱单',
  customs_dec: '报关单',
  co: '产地证',
  insurance: '保险单',
  contract: '合同',
  other: '其他',
};

// 协作状态名称
export const CollaborationStatusNames: Record<CollaborationStatus, string> = {
  invited: '已邀请',
  accepted: '已接受',
  in_progress: '进行中',
  completed: '已完成',
  cancelled: '已取消',
};

// Partner创建请求
export interface PartnerCreateRequest {
  name: string;
  code: string;
  type: PartnerType;
  stage?: LogisticsStage;
  sub_type?: string;
  contact_name?: string;
  phone?: string;
  email?: string;
  address?: string;
  country?: string;
  region?: string;
  service_ports?: string;
  service_routes?: string;
  certifications?: string;
  api_config?: string;
  service_capabilities?: string;
  contract_info?: string;
  payment_terms?: string;
  insurance_coverage?: number;
}

// Partner更新请求
export interface PartnerUpdateRequest {
  name?: string;
  sub_type?: string;
  contact_name?: string;
  phone?: string;
  email?: string;
  address?: string;
  country?: string;
  region?: string;
  service_ports?: string;
  service_routes?: string;
  rating?: number;
  certifications?: string;
  api_config?: string;
  service_capabilities?: string;
  contract_info?: string;
  payment_terms?: string;
  insurance_coverage?: number;
  status?: 'active' | 'inactive';
}

// 协作创建请求
export interface CollaborationCreateRequest {
  partner_id: string;
  role: PartnerType;
  task_desc?: string;
}

// ============ Phase 4: 运价引擎类型 ============

// 集装箱类型
export type ContainerType = '20GP' | '40GP' | '40HQ' | '45HQ' | 'LCL';

// 集装箱类型名称
export const ContainerTypeNames: Record<ContainerType, string> = {
  '20GP': '20尺普柜',
  '40GP': '40尺普柜',
  '40HQ': '40尺高柜',
  '45HQ': '45尺高柜',
  'LCL': '散货/拼箱',
};

// 运价
export interface FreightRate {
  id: number;
  partner_id: string;
  partner_name?: string;
  origin: string;
  origin_name: string;
  destination: string;
  destination_name: string;
  transit_days: number;
  carrier: string;
  container_type: ContainerType;
  currency: string;
  ocean_freight: number;
  baf: number;
  caf: number;
  pss: number;
  gri: number;
  thc: number;
  doc_fee: number;
  seal_fee: number;
  other_fee: number;
  total_fee: number;
  valid_from: string;
  valid_to: string;
  is_active: boolean;
  remarks?: string;
}

// 运价创建请求
export interface RateCreateRequest {
  partner_id: string;
  origin: string;
  origin_name?: string;
  destination: string;
  destination_name?: string;
  transit_days?: number;
  carrier?: string;
  container_type: ContainerType;
  currency?: string;
  ocean_freight?: number;
  baf?: number;
  caf?: number;
  pss?: number;
  gri?: number;
  thc?: number;
  doc_fee?: number;
  seal_fee?: number;
  other_fee?: number;
  valid_from: string;
  valid_to: string;
  remarks?: string;
}

// 比价请求
export interface RateCompareRequest {
  origin: string;
  destination: string;
  container_type?: ContainerType;
  ship_date?: string;
}

// 比价结果
export interface RateCompareResult extends FreightRate {
  rank: number;
  price_diff: number;
  price_diff_pct: number;
  recommendation?: string;
}

// 航线信息
export interface RouteInfo {
  origin: string;
  origin_name: string;
  destination: string;
  destination_name: string;
  options_count: number;
  lowest_price: number;
}

// 货代绩效
export interface PartnerPerformance {
  id: number;
  partner_id: string;
  route_lane: string;
  total_shipments: number;
  on_time_shipments: number;
  delayed_shipments: number;
  on_time_rate: number;
  avg_transit_days: number;
  avg_delay_days: number;
  total_claim: number;
  rating: number;
  period_start: string;
  period_end: string;
}

// ============ Phase 5: 文档自动化扩展类型 ============

// 生成的文档
export interface GeneratedDocument {
  id: number;
  shipment_id: string;
  template_id?: number;
  doc_type: string;
  file_name: string;
  file_path: string;
  file_size: number;
  mime_type: string;
  version: number;
  generated_by: string;
  generated_at: string;
  download_url?: string;
}

// OCR识别结果
export interface OCRResult {
  id: number;
  document_id: number;
  field_name: string;
  field_label: string;
  field_value: string;
  confidence: number;
  applied: boolean;
  bounding_box?: string;
}

// 文档模板
export interface DocumentTemplate {
  id: number;
  code: string;
  name: string;
  template_type: string;
  description?: string;
  is_active: boolean;
}

// 上传单据请求
export interface UploadDocumentRequest {
  file: File;
  doc_type: DocumentType;
  doc_name?: string;
  remarks?: string;
}

// 审核单据请求
export interface ReviewDocumentRequest {
  status: DocumentStatus;
  remarks?: string;
}


export interface SendSMSCodeRequest {
  phone_country_code?: string;
  phone_number: string;
  scene: "login" | "reset_password";
}

export interface SMSLoginRequest {
  phone_country_code?: string;
  phone_number: string;
  code: string;
}

export interface SMSLoginResponse {
  need_select_org: boolean;
  token?: string;
  token_temp?: string;
  user?: User;
  orgs?: UserOrg[];
}
