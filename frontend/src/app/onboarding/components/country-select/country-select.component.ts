import { Component, ViewEncapsulation, ElementRef, EventEmitter, HostListener, Input, Output, ViewChild } from '@angular/core';
import { NgClass } from '@angular/common';

export interface Country {
  code: string;
  name: string;
}

@Component({
  selector: 'app-country-select',
  standalone: true,
  imports: [NgClass],
  templateUrl: './country-select.component.html',
  styleUrl: './country-select.component.css',
  encapsulation: ViewEncapsulation.None
})
export class CountrySelectComponent {
  @Input() countries: Country[] = [];
  @Input() value = '';
  @Input() hasError = false;
  @Output() valueChange = new EventEmitter<string>();

  @ViewChild('searchInput') searchInput!: ElementRef<HTMLInputElement>;

  isOpen = false;
  searchText = '';
  private justOpened = false;

  get selectedCountry(): Country | undefined {
    return this.countries.find(c => c.code === this.value);
  }

  get filteredCountries(): Country[] {
    if (!this.searchText) return this.countries;
    const term = this.searchText.toLowerCase();
    return this.countries.filter(c => c.name.toLowerCase().includes(term));
  }

  getFlagUrl(code: string): string {
    return `https://flagcdn.com/w40/${code.toLowerCase()}.png`;
  }

  open(): void {
    this.isOpen = true;
    this.searchText = '';
    this.justOpened = true;
    setTimeout(() => {
      this.justOpened = false;
      this.searchInput?.nativeElement.focus();
    });
  }

  select(country: Country): void {
    this.value = country.code;
    this.valueChange.emit(country.code);
    this.isOpen = false;
    this.searchText = '';
  }

  @HostListener('document:click', ['$event'])
  onClickOutside(event: Event): void {
    if (this.justOpened) return;
    if (!this.elementRef.nativeElement.contains(event.target)) {
      this.isOpen = false;
      this.searchText = '';
    }
  }

  constructor(private elementRef: ElementRef) {}
}
