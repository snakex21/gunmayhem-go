vx = Math.random() * 10 - 5;
vy = Math.random() * 5 - 15;
firepower = Math.abs(asdf % 1000);
vx = _rotation * firepower;
vy = - firepower;
if(firepower < 0)
{
   vy += firepower;
}
fusetime = 0;
_rotation = _rotation * 50;
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
   if(!_root.GAMEPAUSED)
   {
      if(!hitground)
      {
         _rotation = _rotation + rotationspeed;
      }
      else if(Math.abs(90 - _rotation) < Math.abs(-90 - _rotation))
      {
         _rotation = _rotation + (90 - _rotation) / 5;
      }
      else
      {
         _rotation = _rotation + (-90 - _rotation) / 5;
      }
      if(vy < 24)
      {
         vy += _root.gravity + 0.12;
      }
      dirx = Math.cos((_rotation - 90) * 3.141592653589793 / 180) * 25;
      diry = Math.sin((_rotation - 90) * 3.141592653589793 / 180) * 25;
      _root.CP("fx_dynamite",_X + dirx,_Y + diry,0);
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
                  rotationspeed *= 0.5;
                  vx *= 0.6;
                  vy *= -0.4;
                  hitground = true;
                  break;
               }
               i++;
            }
         }
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
      if(fusetime == 40)
      {
         i = 0;
         while(i < _root.activeplayers.length)
         {
            if(_root.activeplayers[i].AI && _root.activeplayers[i].perknumber != 6)
            {
               distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
               if(_root.activeplayers[i]._y < _Y + 20 && distance <= 150)
               {
                  _root.activeplayers[i].KEYUP = true;
                  if(_root.activeplayers[i]._x < _X)
                  {
                     _root.activeplayers[i].lockright = 10;
                  }
                  if(_root.activeplayers[i]._x > _X)
                  {
                     _root.activeplayers[i].lockleft = 10;
                  }
               }
            }
            i++;
         }
      }
      if(fusetime > 50)
      {
         i = 0;
         while(i < _root.activeplayers.length)
         {
            distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
            if(distance <= 200)
            {
               radians = Math.atan2(_root.activeplayers[i]._y - 30 - _Y,_root.activeplayers[i]._x - _X);
               degrees = radians * 180 / 3.141592653589793;
               if(_root.activeplayers[i].perknumber == 6 || _root.activeplayers[i].shieldtime > 0)
               {
                  pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 25;
                  pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 50;
                  if(_root.activeplayers[i].shield._alpha > 99)
                  {
                     _root.activeplayers[i].shield.gotoAndPlay(23);
                  }
               }
               else
               {
                  pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 5;
                  pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 10;
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
         if(_root._quality != "LOW")
         {
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
         }
         _root.explodesound();
         _root.CP("fx_combo",_X,_Y - 50,0,-2);
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
      if(_Y >= 900 || _X < -500 || _X > 1400 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
