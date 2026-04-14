import { Component, ViewEncapsulation, EventEmitter, Output } from '@angular/core';

@Component({
  selector: 'app-formal-landing',
  standalone: true,
  templateUrl: './formal-landing.component.html',
  encapsulation: ViewEncapsulation.None,
  styles: [`:host { display: block; height: 100%; overflow-y: auto; }`]
})
export class FormalLandingComponent {
  @Output() next = new EventEmitter<void>();
}
