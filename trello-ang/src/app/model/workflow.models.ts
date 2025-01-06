export interface Workflow {
  elementId: string;
  id: number;
  dependency_task: string[];
  is_active: boolean;
  task_id: string;
}
