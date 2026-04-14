import { Component, ViewEncapsulation } from '@angular/core';

@Component({
  selector: 'app-success',
  standalone: true,
  templateUrl: './success.component.html',
  styleUrl: './success.component.css',
  encapsulation: ViewEncapsulation.None
})
export class SuccessComponent {
  goHome(): void {
    window.location.href = 'https://dome-marketplace-sbx.org/';
  }
}
