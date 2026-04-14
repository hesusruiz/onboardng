import { Component, ViewEncapsulation } from '@angular/core';

@Component({
  selector: 'app-formal-thank-you',
  standalone: true,
  templateUrl: './formal-thank-you.component.html',
  styleUrl: './formal-thank-you.component.css',
  encapsulation: ViewEncapsulation.None
})
export class FormalThankYouComponent {
  goHome(): void {
    window.location.href = 'https://dome-marketplace-sbx.org/';
  }
}
