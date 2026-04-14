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
  @Input() termsUrl = 'https://onboard.dome.mycredential.eu/api/files/pbc_365946868/ki9zjz96v7re63n/20250303_dome_t_c_for_cloud_customers_6zqyb3yz7d.pdf';
  @Output() next = new EventEmitter<void>();
  @Output() back = new EventEmitter<void>();
}
