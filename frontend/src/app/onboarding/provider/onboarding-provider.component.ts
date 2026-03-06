import { Component, ViewEncapsulation } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ProviderLandingComponent } from './components/landing/provider-landing.component';
import { TermsComponent } from '../components/terms/terms.component';
import { RepresentativeComponent } from '../components/representative/representative.component';
import { CompanyInfoComponent } from '../components/company-info/company-info.component';
import { EmailVerificationComponent } from '../components/email-verification/email-verification.component';
import { ProviderSuccessComponent } from './components/success/provider-success.component';

@Component({
  selector: 'app-onboarding-provider',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    ProviderLandingComponent,
    TermsComponent,
    RepresentativeComponent,
    CompanyInfoComponent,
    EmailVerificationComponent,
    ProviderSuccessComponent
  ],
  templateUrl: './onboarding-provider.component.html',
  styleUrl: './onboarding-provider.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingProviderComponent {
  currentStep = 0;

  onboardingForm = new FormGroup({
    meetsRequirements: new FormControl(false, [Validators.requiredTrue]),
    acceptTerms: new FormControl(false, [Validators.requiredTrue]),
    representative: new FormGroup({
      firstName: new FormControl('', [Validators.required]),
      lastName: new FormControl('', [Validators.required]),
      email: new FormControl('', [Validators.required, Validators.email]),
      isAuthorised: new FormControl(false, [Validators.requiredTrue]),
    }),
    company: new FormGroup({
      legalName: new FormControl('', [Validators.required]),
      vatNumber: new FormControl('', [Validators.required]),
      country: new FormControl('', [Validators.required]),
      city: new FormControl('', [Validators.required]),
      street: new FormControl('', [Validators.required]),
      postalCode: new FormControl('', [Validators.required]),
    }),
  });

  get meetsRequirementsControl(): FormControl {
    return this.onboardingForm.get('meetsRequirements') as FormControl;
  }

  get acceptTermsControl(): FormControl {
    return this.onboardingForm.get('acceptTerms') as FormControl;
  }

  get representativeGroup(): FormGroup {
    return this.onboardingForm.get('representative') as FormGroup;
  }

  get companyGroup(): FormGroup {
    return this.onboardingForm.get('company') as FormGroup;
  }

  goNext(): void {
    this.currentStep++;
  }

  goBack(): void {
    this.currentStep--;
  }

  onComplete(): void {
    // TODO: Submit registration data to API
    this.currentStep = 4;
  }

  onVerify(code: string): void {
    // TODO: Verify OTP code via API
    this.currentStep = 5;
  }
}
