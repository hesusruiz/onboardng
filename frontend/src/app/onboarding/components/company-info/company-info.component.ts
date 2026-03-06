import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormGroup, ReactiveFormsModule } from '@angular/forms';
import { NgClass } from '@angular/common';
import { ProgressBarComponent } from '../progress-bar/progress-bar.component';
import { StepFooterComponent } from '../step-footer/step-footer.component';
import { CountrySelectComponent, Country } from '../country-select/country-select.component';

@Component({
  selector: 'app-company-info',
  standalone: true,
  imports: [ReactiveFormsModule, NgClass, ProgressBarComponent, StepFooterComponent, CountrySelectComponent],
  templateUrl: './company-info.component.html',
  styleUrl: './company-info.component.css',
  encapsulation: ViewEncapsulation.None
})
export class CompanyInfoComponent {
  @Input() formGroup!: FormGroup;
  @Input() isLoading = false;
  @Input() errorMessage = '';
  @Output() complete = new EventEmitter<void>();
  @Output() back = new EventEmitter<void>();

  onComplete(): void {
    if (this.formGroup.invalid) {
      this.formGroup.markAllAsTouched();
      return;
    }
    this.complete.emit();
  }

  countries: Country[] = [
    { code: 'AT', name: 'Austria' },
    { code: 'BE', name: 'Belgium' },
    { code: 'BG', name: 'Bulgaria' },
    { code: 'HR', name: 'Croatia' },
    { code: 'CY', name: 'Cyprus' },
    { code: 'CZ', name: 'Czech Republic' },
    { code: 'DK', name: 'Denmark' },
    { code: 'EE', name: 'Estonia' },
    { code: 'FI', name: 'Finland' },
    { code: 'FR', name: 'France' },
    { code: 'DE', name: 'Germany' },
    { code: 'GR', name: 'Greece' },
    { code: 'HU', name: 'Hungary' },
    { code: 'IE', name: 'Ireland' },
    { code: 'IT', name: 'Italy' },
    { code: 'LV', name: 'Latvia' },
    { code: 'LT', name: 'Lithuania' },
    { code: 'LU', name: 'Luxembourg' },
    { code: 'MT', name: 'Malta' },
    { code: 'NL', name: 'Netherlands' },
    { code: 'PL', name: 'Poland' },
    { code: 'PT', name: 'Portugal' },
    { code: 'RO', name: 'Romania' },
    { code: 'SK', name: 'Slovakia' },
    { code: 'SI', name: 'Slovenia' },
    { code: 'ES', name: 'Spain' },
    { code: 'SE', name: 'Sweden' },
  ];
}
