import { Component, Input } from '@angular/core';
import { Project } from '../../model/project.model';

@Component({
  selector: 'app-project-details',
  templateUrl: './project-details.component.html',
  styleUrls: ['./project-details.component.css']
})
export class ProjectDetailsComponent {
  @Input() project: Project | null = null;
}
