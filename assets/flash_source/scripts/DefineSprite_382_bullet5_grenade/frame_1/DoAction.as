function blowup()
{
   switch(Math.floor(asdf / 1000))
   {
      case 1:
         immunity = 1;
         break;
      case 2:
         immunity = 2;
         break;
      case 3:
         immunity = 3;
         break;
      case 4:
         immunity = 4;
   }
   i = 0;
   while(i < _root.activeplayers.length)
   {
      distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
      if(distance <= 200)
      {
         radians = Math.atan2(_root.activeplayers[i]._y - 30 - _Y,_root.activeplayers[i]._x - _X);
         degrees = radians * 180 / 3.141592653589793;
         pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 5;
         pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 10;
         if(_root.activeplayers[i].PLAYERNUMBER == immunity)
         {
            pushx *= 0.5;
            pushy *= 0.5;
         }
         _root.activeplayers[i].vx += pushx;
         _root.activeplayers[i].vy += pushy;
         switch(Math.floor(asdf / 1000))
         {
            case 1:
               _root.activeplayers[i].gothitby = _root.p1name;
               break;
            case 2:
               _root.activeplayers[i].gothitby = _root.p2name;
               break;
            case 3:
               _root.activeplayers[i].gothitby = _root.p3name;
               break;
            case 4:
               _root.activeplayers[i].gothitby = _root.p4name;
         }
         _root.activeplayers[i].hitbynade = true;
      }
      i++;
   }
   shake = random(30) + 20;
   if(random(2) == 0)
   {
      shake *= -1;
   }
   _root._x += shake;
   _root.CP("fx_ex6",_X,_Y - 20,0);
   _root.CP("fx_ex2",_X,_Y - 20,0);
   _root.CP("fx_ex",_X,_Y - 20,0);
   _root.CP("fx_ex4",_X,_Y - 20,0);
   i = 0;
   while(i < 5)
   {
      _root.CP("fx_ex5",_X,_Y - 20,0);
      i++;
   }
   i = 0;
   while(i < 5)
   {
      _root.CP("fx_ex3",_X,_Y - 20,0);
      i++;
   }
   _root.CP("fx_combo",_X,_Y - 50,0,-2);
   removeMovieClip(this);
   delete this.onEnterFrame;
}
stop();
firepower = Math.abs(asdf % 1000);
firepower = 20;
vx = _rotation * firepower;
vy = -8;
if(firepower < 0)
{
   vy += firepower;
}
fusetime = 0;
_rotation = random(360);
rotationspeed = Math.random() * 10 + 5;
if(random(2) == 0)
{
   rotationspeed *= -1;
}
freepass = false;
hitground = false;
_xscale = 80;
_yscale = 80;
this.onEnterFrame = function()
{
   _rotation = _rotation + rotationspeed;
   if(vy < 24)
   {
      vy += _root.gravity + 0.12;
   }
   _X = _X + vx;
   _Y = _Y + vy;
   if(!freepass)
   {
      if(_root.ground.hitTest(_X,_Y + 8,true) && vy > 0)
      {
         _Y = _Y - vy;
         i = 1;
         while(i <= 5)
         {
            if(_root.ground.hitTest(_X,_Y + 8 + i * (vy / 5),true))
            {
               _Y = _Y + vy / 5 * (i - 1);
               vx *= 0.8;
               vy *= -0.8;
               break;
            }
            i++;
         }
      }
   }
   i = 0;
   while(i < _root.activeplayers.length)
   {
      distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
      if(distance < 40)
      {
         blowup();
      }
      i++;
   }
   if(!_root.ground.hitTest(_X,_Y,true) && !_root.ground.hitTest(_X,_Y + 10,true))
   {
      freepass = false;
   }
   if(_root.ground.hitTest(_X,_Y,true) && !freepass)
   {
      freepass = true;
   }
   fusetime += 1;
   if(fusetime > 40)
   {
      blowup();
   }
   if(_Y >= 900 || _X < -500 || _X > 1400 || _root.deleteeverything)
   {
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
};
