import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { ProgressBarComponent } from '../progress-bar/progress-bar.component';
import { StepFooterComponent } from '../step-footer/step-footer.component';

@Component({
  selector: 'app-terms',
  standalone: true,
  imports: [ReactiveFormsModule, ProgressBarComponent, StepFooterComponent],
  templateUrl: './terms.component.html',
  styleUrl: './terms.component.css',
  encapsulation: ViewEncapsulation.None
})
export class TermsComponent {
  @Input() acceptTermsControl!: FormControl;
  @Input() termsLabel = 'DOME Terms and Conditions for Customers';
  @Input() termsUrl = 'https://dome-marketplace.eu/assets/documents/terms.pdf';
  @Output() next = new EventEmitter<void>();
  @Output() back = new EventEmitter<void>();
}
