import { Component, ViewEncapsulation, inject, ChangeDetectorRef } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { finalize } from 'rxjs';
import { latinOnly } from '../validators/latin-only.validator';
import { RegistrationContextService } from '../services/registration-context.service';
import { FormalLandingComponent } from './components/landing/formal-landing.component';
import { LegalRepresentativeComponent } from './components/legal-representative/legal-representative.component';
import { LearComponent } from './components/lear/lear.component';
import { ContractualDocumentsComponent } from './components/contractual-documents/contractual-documents.component';
import { FormalThankYouComponent } from './components/thank-you/formal-thank-you.component';

@Component({
  selector: 'app-onboarding-provider-formal',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    FormalLandingComponent,
    LegalRepresentativeComponent,
    LearComponent,
    ContractualDocumentsComponent,
    FormalThankYouComponent,
  ],
  templateUrl: './onboarding-provider-formal.component.html',
  styleUrl: './onboarding-provider-formal.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingProviderFormalComponent {
  private http = inject(HttpClient);
  private cdr = inject(ChangeDetectorRef);
  private registrationContext = inject(RegistrationContextService);

  currentStep = 0;
  isSubmitting = false;
  submitError = '';

  formalForm = new FormGroup({
    legalRepresentative: new FormGroup({
      firstName: new FormControl('', [Validators.required, latinOnly]),
      lastName: new FormControl('', [Validators.required, latinOnly]),
      email: new FormControl('', [Validators.required, Validators.email]),
      nationality: new FormControl(''),
      idCardNumber: new FormControl('', [latinOnly]),
    }),
    lear: new FormGroup({
      firstName: new FormControl('', [Validators.required, latinOnly]),
      lastName: new FormControl('', [Validators.required, latinOnly]),
      email: new FormControl('', [Validators.required, Validators.email]),
      nationality: new FormControl(''),
      professionalAddress: new FormControl('', [latinOnly]),
      idCardNumber: new FormControl('', [latinOnly]),
      mobilePhone: new FormControl('', [latinOnly]),
      isAuthorised: new FormControl(false, [Validators.requiredTrue]),
    }),
    contractualDocuments: new FormGroup({
      hasEidasCertificate: new FormControl<boolean | null>(null, [Validators.required]),
    }),
  });

  get legalRepGroup(): FormGroup {
    return this.formalForm.get('legalRepresentative') as FormGroup;
  }

  get learGroup(): FormGroup {
    return this.formalForm.get('lear') as FormGroup;
  }

  get contractualGroup(): FormGroup {
    return this.formalForm.get('contractualDocuments') as FormGroup;
  }

  goNext(): void {
    this.currentStep++;
  }

  goBack(): void {
    this.currentStep--;
  }

  onComplete(): void {
    this.isSubmitting = true;
    this.submitError = '';

    const context = this.registrationContext.getContext();
    const legalRep = this.legalRepGroup.value;
    const lear = this.learGroup.value;

    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    });

    const body = {
      vatId: context.vatId,
      email: context.email,
      code: context.code,
      lr_first_name: legalRep.firstName,
      lr_last_name: legalRep.lastName,
      lr_email: legalRep.email,
      lr_country: legalRep.nationality,
      lr_id_card: legalRep.idCardNumber,
      lear_first_name: lear.firstName,
      lear_last_name: lear.lastName,
      lear_email: lear.email,
      lear_country: lear.nationality,
      lear_address: lear.professionalAddress,
      lear_id_card: lear.idCardNumber,
      lear_mobile_number: lear.mobilePhone,
    };

    this.http.post<any>('/api/representatives', body, { headers })
      .pipe(finalize(() => {
        this.isSubmitting = false;
        this.cdr.detectChanges();
      }))
      .subscribe({
        next: (res: any) => {
          if (res && res.success === true) {
            this.registrationContext.clearContext();
            this.currentStep = 4;
          } else {
            this.submitError = res?.message || 'Submission failed. Please try again.';
          }
          this.cdr.detectChanges();
        },
        error: (err: any) => {
          this.submitError = err.error?.message || 'An error occurred. Please try again.';
          this.cdr.detectChanges();
        }
      });
  }
}
