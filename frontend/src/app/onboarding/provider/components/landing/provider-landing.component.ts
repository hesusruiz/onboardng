import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { NgClass } from '@angular/common';

@Component({
  selector: 'app-provider-landing',
  standalone: true,
  imports: [ReactiveFormsModule, NgClass],
  templateUrl: './provider-landing.component.html',
  encapsulation: ViewEncapsulation.None,
  styles: [`:host { display: block; height: 100%; overflow-y: auto; }`]
})
export class ProviderLandingComponent {
  @Input() meetsRequirementsControl!: FormControl;
  @Output() next = new EventEmitter<void>();
}
