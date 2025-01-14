import { Injectable } from '@angular/core';
import { CanActivate, ActivatedRouteSnapshot, Router } from '@angular/router';

@Injectable({
  providedIn: 'root',
})
export class AuthGuard implements CanActivate {
  constructor(private router: Router) {}

  canActivate(route: ActivatedRouteSnapshot): boolean {
    const isAuthenticated = !!localStorage.getItem('access_token');
    const userRole = localStorage.getItem('role') || '';
    const allowedRoles = route.data['roles'] as Array<string>;

    if (isAuthenticated && allowedRoles.includes(userRole)) {
      return true;
    } else {
      this.router.navigate(['/login']);
      return false;
    }
  }
}
