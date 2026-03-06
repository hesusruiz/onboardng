import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormGroup, ReactiveFormsModule } from '@angular/forms';
import { NgClass } from '@angular/common';
import { ProgressBarComponent } from '../progress-bar/progress-bar.component';
import { StepFooterComponent } from '../step-footer/step-footer.component';

@Component({
  selector: 'app-representative',
  standalone: true,
  imports: [ReactiveFormsModule, NgClass, ProgressBarComponent, StepFooterComponent],
  templateUrl: './representative.component.html',
  styleUrl: './representative.component.css',
  encapsulation: ViewEncapsulation.None
})
export class RepresentativeComponent {
  @Input() formGroup!: FormGroup;
  @Input() confirmText = 'I confirm I am authorised to use this account for procurement-related actions.';
  @Input() isLoading = false;
  @Input() errorMessage = '';
  @Output() next = new EventEmitter<void>();
  @Output() back = new EventEmitter<void>();

  onNext(): void {
    if (this.formGroup.invalid) {
      this.formGroup.markAllAsTouched();
      return;
    }
    this.next.emit();
  }
}
