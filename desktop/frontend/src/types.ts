export type {
  AssessRequest, AssessResponse,
  CompareRequest, CompareResponse,
  ValidateRequest, ValidateResponse,
  CheckPatternsRequest, CheckPatternsResponse,
  MetricResult, PatternResult, TaskDelta,
} from '../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/models.js'

export interface DCMAMetricDef {
  id: number
  name: string
  selected: boolean
}
