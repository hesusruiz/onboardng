import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { NgClass } from '@angular/common';

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [ReactiveFormsModule, NgClass],
  templateUrl: './landing.component.html',
  styleUrl: './landing.component.css',
  encapsulation: ViewEncapsulation.None
})
export class LandingComponent {
  @Input() isCompanyControl!: FormControl;
  @Output() next = new EventEmitter<void>();
}
